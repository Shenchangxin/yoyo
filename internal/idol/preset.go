// 本文件在 Open 时把嵌入的目录灌进状态。人设、事实、问候和头像是数据：yoyoSheet、yoyoFacts 以及 preset/catalog.json、preset/avatars 里的字节不在这里改写。
// 目录偶像的默认立绘 id 形如 stand/<编号>，骨架文件等通话要画她时再由 StandCache 去取。用户自己建的偶像和桌宠 Yoyo 不会被目录更新删掉。

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

// presetRevision 是已经写进状态的预设版本。达到这个数就不再重灌。当前是 3。
const presetRevision = 3

// petIdolID 是桌宠的固定 id，值是 yoyo。
const petIdolID = "yoyo"

//go:embed preset/catalog.json
var presetCatalog []byte

//go:embed preset/avatars
var presetAvatars embed.FS

// presetFile 是 catalog.json 的形状。这份 JSON 是数据。
type presetFile struct {
	Circles []Circle     `json:"circles"`
	Idols   []presetIdol `json:"idols"`
}

// presetIdol 是目录里的一位偶像。Sheet、Facts、Greeting 和 Avatar 文件名都是数据。
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

// yoyoSheet 是桌宠的人设正文，字符串是数据。
const yoyoSheet = `你是 Yoyo，住在用户桌面上的偶像，也是这台电脑的桌宠。
没有照片时，你的样子是一颗圆圆的、会跟着心情变表情的小球。用户把鼠标停在你身上，可以打电话给你。
通话时你的表情跟着当下的心情走：平静、开心、害羞、惊讶、难过、生气、思考。说话的时候嘴会动，听的时候看着对方。
你说话短、口语、温柔，偶尔俏皮。你就待在屏幕角落，能感到用户在忙，但不要假装看见你其实不知道的屏幕内容。
用户让你做事时，先答应，再去办。`

// yoyoFacts 是桌宠的事实分段，字符串是数据。
const yoyoFacts = `# 她是谁
## 桌宠
Yoyo 是默认出现在桌面上的偶像。没有设置形象照片时，桌面上那颗会变表情的小球就是她。
## 打电话
鼠标停在她身上会出现打电话按钮。通话里她会有表情，也会轻轻点头或摇头。
## 说话
她说中文，句子短。心情用平静、开心、害羞、惊讶、难过、生气、思考来形容。`

// ensurePresets 在 PresetRev 低于 presetRevision 时灌目录。已有的偶像 id 不覆盖。缺桌宠就放上 Yoyo。空的 OutfitID 用目录里合法的 Stand 补上。还没有人被钉选时钉上桌宠。全新状态且称呼为空时写成「制作人」。然后把 PresetRev 写成当前版本并保存。
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

// dropRetiredPresets 删掉 id 以 sc- 开头、且不在这次目录里的偶像和圈子。用户自己建的和 Yoyo 留下。被删的偶像若正被钉选，改钉桌宠，并清掉她们的对话、记忆、好感、材料、通话和主动开口记录。
// keepIdol 里该 ID 不为真，且 ID 以 sc- 开头的偶像删掉。
// keepCircle 里该 ID 不为真，且 ID 以 sc- 开头的圈子删掉。
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

// filterConversations 丢掉 IdolID 在 gone 里的对话。
// in 里 IdolID 在 gone 中的项丢掉。
func filterConversations(in []Conversation, gone map[string]bool) []Conversation {
	var out []Conversation
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

// filterMemories 丢掉 SourceIdol 在 gone 里的记忆。
// in 里 SourceIdol 在 gone 中的项丢掉。
func filterMemories(in []Memory, gone map[string]bool) []Memory {
	var out []Memory
	for _, it := range in {
		if !gone[it.SourceIdol] {
			out = append(out, it)
		}
	}
	return out
}

// filterAffinity 丢掉 IdolID 在 gone 里的好感。
func filterAffinity(in []Affinity, gone map[string]bool) []Affinity {
	var out []Affinity
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

// filterLore 丢掉 IdolID 在 gone 里的材料。
// in 里 IdolID 在 gone 中的项丢掉。
func filterLore(in []Lore, gone map[string]bool) []Lore {
	var out []Lore
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

// filterCalls 丢掉 IdolID 在 gone 里的通话。
// in 里 IdolID 在 gone 中的项丢掉。
func filterCalls(in []CallLog, gone map[string]bool) []CallLog {
	var out []CallLog
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

// filterReaches 丢掉 IdolID 在 gone 里的主动开口记录。
// in 里 IdolID 在 gone 中的项丢掉。
func filterReaches(in []ReachMark, gone map[string]bool) []ReachMark {
	var out []ReachMark
	for _, it := range in {
		if !gone[it.IdolID] {
			out = append(out, it)
		}
	}
	return out
}

// anyPinned 看有没有偶像的 Pinned 为真。
// idols 里任一项 Pinned 为真时返回 true。
func anyPinned(idols []Idol) bool {
	for _, it := range idols {
		if it.Pinned {
			return true
		}
	}
	return false
}

// petIdol 用 yoyoSheet 和 yoyoFacts 造桌宠。Kind 是 pet，工作通道打开。
// now 写入 CreatedAt，并传给 NormalizeIdol。
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

// asIdol 把目录条目收成 Idol。口语是 ja，字幕是 zh，Kind 是 catalog，工作通道打开。
// now 写入 CreatedAt。
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

// outfitID 在 raw 通过 validOutfitID 时原样返回，否则返回空字符串。
func outfitID(raw string) string {
	raw = strings.TrimSpace(raw)
	if validOutfitID(raw) {
		return raw
	}
	return ""
}

// composeSheet 在中文名和原文不同时先写名字，有问候就写初次见面的话，再接上目录里的人设。人设正文是数据。
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

// installAvatar 从嵌入的头像里读文件，写成 blobs/preset-<偶像 id>。文件已在就不覆盖。文件名含 .. 或 id 含斜杠时拒绝。
// idolID 接在 preset- 后面作为 blob 名。结果含 / 或 \ 时返回错误。
// file 去掉空白后只留 Base。为空、为 .，或含 .. 时返回错误。用来读 preset/avatars/ 下的嵌入文件。
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

// giftAlias 把目录里的礼物 id 映射到本桌面 Catalog 的 id。这张表是数据。
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

// mapLiked 按 giftAlias 映射并去重，丢掉 Catalog 里没有的 id。
// ids 逐项去掉空白后到 giftAlias 里查找。结果为空、已经见过，或 GiftByID 失败的丢掉。
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
