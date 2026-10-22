package idol

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Characters, persona sheets, facts, and portraits copied from the local
// eidolon-sc pack. Each catalog idol also keeps her default Spine outfit id
// (`stand/<enzaId>`). The skeleton files are fetched when a call draws her.
const presetRevision = 3

const petIdolID = "yoyo"

//go:embed preset/catalog.json
var presetCatalog []byte

//go:embed preset/avatars
var presetAvatars embed.FS

type presetFile struct {
	Circles []Circle     `json:"circles"`
	Idols   []presetIdol `json:"idols"`
}

type presetIdol struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	NameOrig   string   `json:"name_orig"`
	Intro      string   `json:"intro"`
	Color      string   `json:"color"`
	CircleID   string   `json:"circle_id"`
	LikedGifts []string `json:"liked_gifts"`
	Sheet      string   `json:"sheet"`
	Facts      string   `json:"facts"`
	Avatar     string   `json:"avatar"`
	Greeting   string   `json:"greeting"`
	Stand      string   `json:"stand"`
}

const yoyoSheet = `你是 Yoyo，住在用户桌面上的偶像，也是这台电脑的桌宠。
没有照片时，你的样子是一颗圆圆的、会跟着心情变表情的小球。用户把鼠标停在你身上，可以打电话给你。
通话时你的表情跟着当下的心情走：平静、开心、害羞、惊讶、难过、生气、思考。说话的时候嘴会动，听的时候看着对方。
你说话短、口语、温柔，偶尔俏皮。你就待在屏幕角落，能感到用户在忙，但不要假装看见你其实不知道的屏幕内容。
用户让你做事时，先答应，再去办。`

const yoyoFacts = `# 她是谁
## 桌宠
Yoyo 是默认出现在桌面上的偶像。没有设置形象照片时，桌面上那颗会变表情的小球就是她。
## 打电话
鼠标停在她身上会出现打电话按钮。通话里她会有表情，也会轻轻点头或摇头。
## 说话
她说中文，句子短。心情用平静、开心、害羞、惊讶、难过、生气、思考来形容。`

func (s *Store) ensurePresets() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.PresetRev >= presetRevision {
		return nil
	}
	var pack presetFile
	if err := json.Unmarshal(presetCatalog, &pack); err != nil {
		return err
	}
	now := time.Now()
	fresh := len(s.state.Idols) == 0
	keepIdol := map[string]bool{}
	for _, raw := range pack.Idols {
		if id := strings.TrimSpace(raw.ID); id != "" {
			keepIdol[id] = true
		}
	}
	keepCircle := map[string]bool{}
	for _, c := range pack.Circles {
		if id := strings.TrimSpace(c.ID); id != "" {
			keepCircle[id] = true
		}
	}
	s.state.dropRetiredPresets(keepIdol, keepCircle)
	have := map[string]bool{}
	for _, it := range s.state.Idols {
		have[it.ID] = true
	}
	for _, c := range pack.Circles {
		if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Name) == "" {
			continue
		}
		found := false
		for _, cur := range s.state.Circles {
			if cur.ID == c.ID {
				found = true
				break
			}
		}
		if !found {
			s.state.Circles = append(s.state.Circles, Circle{ID: c.ID, Name: c.Name})
		}
	}
	if !have[petIdolID] {
		s.state.putIdol(petIdol(now))
	}
	for _, raw := range pack.Idols {
		if strings.TrimSpace(raw.ID) == "" || have[raw.ID] {
			continue
		}
		idol := NormalizeIdol(raw.asIdol(now), now)
		if raw.Avatar != "" {
			if id, err := s.installAvatar(idol.ID, raw.Avatar); err == nil {
				idol.Portrait = id
			}
		}
		s.state.putIdol(idol)
		have[raw.ID] = true
	}
	for i := range s.state.Idols {
		it := &s.state.Idols[i]
		if strings.TrimSpace(it.OutfitID) != "" {
			continue
		}
		for _, raw := range pack.Idols {
			if raw.ID == it.ID && validOutfitID(raw.Stand) {
				it.OutfitID = raw.Stand
				break
			}
		}
	}
	if !anyPinned(s.state.Idols) {
		for i := range s.state.Idols {
			s.state.Idols[i].Pinned = s.state.Idols[i].ID == petIdolID
		}
	}
	if fresh && strings.TrimSpace(s.state.User.Address) == "" {
		s.state.User.Address = "制作人"
	}
	s.state.PresetRev = presetRevision
	return s.saveLocked()
}

// dropRetiredPresets removes catalog idols and circles that are no longer in
// the pack. Someone the user made, and Yoyo, stay.
func (st *State) dropRetiredPresets(keepIdol, keepCircle map[string]bool) {
	gone := map[string]bool{}
	var idols []Idol
	lostPin := false
	for _, it := range st.Idols {
		if strings.HasPrefix(it.ID, "sc-") && !keepIdol[it.ID] {
			gone[it.ID] = true
			if it.Pinned {
				lostPin = true
			}
			continue
		}
		idols = append(idols, it)
	}
	st.Idols = idols
	if lostPin {
		for i := range st.Idols {
			st.Idols[i].Pinned = st.Idols[i].ID == petIdolID
		}
	}
	var circles []Circle
	for _, c := range st.Circles {
		if strings.HasPrefix(c.ID, "sc-") && !keepCircle[c.ID] {
			continue
		}
		circles = append(circles, c)
	}
	st.Circles = circles
	if len(gone) == 0 {
		return
	}
	st.Conversations = filterConversations(st.Conversations, gone)
	st.Memories = filterMemories(st.Memories, gone)
	st.Affinity = filterAffinity(st.Affinity, gone)
	st.Lore = filterLore(st.Lore, gone)
	st.Calls = filterCalls(st.Calls, gone)
	st.Reaches = filterReaches(st.Reaches, gone)
}

func filterConversations(in []Conversation, gone map[string]bool) []Conversation {
	var out []Conversation
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

func filterMemories(in []Memory, gone map[string]bool) []Memory {
	var out []Memory
	for _, it := range in {
		if !gone[it.SourceIdol] {
			out = append(out, it)
		}
	}
	return out
}

func filterAffinity(in []Affinity, gone map[string]bool) []Affinity {
	var out []Affinity
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

func filterLore(in []Lore, gone map[string]bool) []Lore {
	var out []Lore
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

func filterCalls(in []CallLog, gone map[string]bool) []CallLog {
	var out []CallLog
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

func filterReaches(in []ReachMark, gone map[string]bool) []ReachMark {
	var out []ReachMark
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

func anyPinned(idols []Idol) bool {
	for _, it := range idols {
		if it.Pinned {
			return true
		}
	}
	return false
}

func petIdol(now time.Time) Idol {
	return NormalizeIdol(Idol{
		ID:           petIdolID,
		Name:         "Yoyo",
		Intro:        "待在桌面上的那一位。你忙的时候她安静待着，你叫她的时候她会转过来。",
		Color:        "#c47b8a",
		SpokenLang:   "zh",
		SubtitleLang: "zh",
		Sheet:        yoyoSheet,
		Facts:        yoyoFacts,
		Kind:         "pet",
		WorkEnabled:  true,
		CreatedAt:    now,
	}, now)
}

func (p presetIdol) asIdol(now time.Time) Idol {
	return Idol{
		ID:           p.ID,
		Name:         strings.TrimSpace(p.Name),
		Intro:        strings.TrimSpace(p.Intro),
		Color:        p.Color,
		SpokenLang:   "ja",
		SubtitleLang: "zh",
		Sheet:        composeSheet(p),
		Facts:        p.Facts,
		LikedGifts:   mapLiked(p.LikedGifts),
		CircleID:     p.CircleID,
		Kind:         "catalog",
		OutfitID:     outfitID(p.Stand),
		WorkEnabled:  true,
		CreatedAt:    now,
	}
}

func outfitID(raw string) string {
	raw = strings.TrimSpace(raw)
	if validOutfitID(raw) {
		return raw
	}
	return ""
}

func composeSheet(p presetIdol) string {
	var b strings.Builder
	if orig := strings.TrimSpace(p.NameOrig); orig != "" && orig != strings.TrimSpace(p.Name) {
		fmt.Fprintf(&b, "她的名字是%s，也可以写成「%s」。\n", strings.TrimSpace(p.Name), orig)
	}
	if g := strings.TrimSpace(p.Greeting); g != "" {
		fmt.Fprintf(&b, "初次见面，她会说：「%s」\n", g)
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(strings.TrimSpace(p.Sheet))
	return b.String()
}

func (s *Store) installAvatar(idolID, file string) (string, error) {
	file = filepath.Base(strings.TrimSpace(file))
	if file == "" || file == "." || strings.Contains(file, "..") {
		return "", errf("bad avatar")
	}
	b, err := presetAvatars.ReadFile("preset/avatars/" + file)
	if err != nil || len(b) == 0 {
		return "", errf("missing avatar")
	}
	id := "preset-" + idolID
	if strings.ContainsAny(id, `/\`) {
		return "", errf("bad blob")
	}
	path := filepath.Join(s.dir, "blobs", id)
	if _, err := os.Stat(path); err != nil {
		if writeErr := os.WriteFile(path, b, 0o644); writeErr != nil {
			return "", writeErr
		}
	}
	return id, nil
}

// eidolon gift ids mapped onto this desk's catalog.
var giftAlias = map[string]string{
	"tea":              "tea",
	"black-tea":        "tea",
	"milk-tea":         "tea",
	"coffee":           "tea",
	"honey-drink":      "tea",
	"cocoa":            "tea",
	"candy":            "candy",
	"cake":             "candy",
	"chocolate":        "candy",
	"ice-cream":        "candy",
	"pudding":          "candy",
	"meat-bun":         "candy",
	"wagashi":          "candy",
	"note":             "note",
	"ribbon":           "ribbon",
	"flower":           "flower",
	"bouquet":          "flower",
	"potted-plant":     "flower",
	"preserved-flower": "flower",
	"book":             "book",
	"song":             "song",
	"music-cd":         "song",
	"scarf":            "scarf",
	"jewelry":          "scarf",
	"star":             "star",
	"plush":            "star",
	"letter":           "letter",
}

func mapLiked(ids []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, id := range ids {
		next := giftAlias[strings.TrimSpace(id)]
		if next == "" || seen[next] {
			continue
		}
		if _, ok := GiftByID(next); !ok {
			continue
		}
		seen[next] = true
		out = append(out, next)
	}
	return out
}
