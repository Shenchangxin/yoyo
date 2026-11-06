// 短片配乐。文件放在视频目录的 bgm 下，选中后抄进内容库；启用了配乐连接时也能按短片的语气生成一段，生成的曲子同时存进目录。
// 一键成片会自动配乐，除非操作者明确说过这条不要配乐。

package video

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// StudioBGMItem 是目录里的一个配乐文件。
type StudioBGMItem struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

// StudioBGMList 是配乐目录和里面的文件。
type StudioBGMList struct {
	Dir   string          `json:"dir"`
	Items []StudioBGMItem `json:"items"`
}

// bgmDir 返回视频目录下的 bgm 文件夹。
func (e *Engine) bgmDir() string {
	return filepath.Join(e.Dir, "bgm")
}

// ListStudioBGM 列出配乐文件，并保证目录存在。只认常见音频扩展名，小于 32 字节的跳过。按文件名排序。没有时条目是空切片。
func (e *Engine) ListStudioBGM() StudioBGMList {
	dir := e.bgmDir()
	_ = os.MkdirAll(dir, 0o755)
	out := StudioBGMList{Dir: dir, Items: []StudioBGMItem{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, ent := range entries {
		if ent.IsDir() || !bgmExt(ent.Name()) {
			continue
		}
		info, err := ent.Info()
		if err != nil || info.Size() < 32 {
			continue
		}
		out.Items = append(out.Items, StudioBGMItem{Name: ent.Name(), Bytes: info.Size()})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Name < out.Items[j].Name })
	return out
}

// bgmExt 判断扩展名是不是 mp3、m4a、wav、aac、flac 或 ogg。
// name 的扩展名转成小写后，与 .mp3、.m4a、.wav、.aac、.flac、.ogg 比较。
func bgmExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".wav", ".aac", ".flac", ".ogg":
		return true
	default:
		return false
	}
}

// AttachStudioBGM 把选中的配乐抄进内容库并写上工程。目录是空的就拒绝。名字空着或是 random 就随机挑一个。名字只能是文件名，不能带路径。超过 20MB 拒绝。音量还没设时写成 0.22。
// id 传给 GetStudioProject。
// name 去掉首尾空白，空串或等于 random 时改成列表里随机一项的 Name，再取 filepath.Base。
func (e *Engine) AttachStudioBGM(id, name string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	list := e.ListStudioBGM()
	if len(list.Items) == 0 {
		return p, fmt.Errorf("把 mp3 放进 %s 后再选", list.Dir)
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "random" {
		name = list.Items[rand.Intn(len(list.Items))].Name
	}
	name = filepath.Base(name)
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return p, fmt.Errorf("bad bgm name")
	}
	path := filepath.Join(list.Dir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return p, fmt.Errorf("bgm not found")
	}
	if len(raw) > 20<<20 {
		return p, fmt.Errorf("bgm is larger than 20MB")
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	return e.setStudioBGM(id, hash, name)
}

// setStudioBGM 把配乐写上工程，并取消「不要配乐」。音量还没设时写成 0.22，垫在口播下面不抢人声。
func (e *Engine) setStudioBGM(id, hash, label string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		p.BGMHash = hash
		p.Storyboard.BGM = hash
		p.NoBGM = false
		p.BGMName = label
		if p.Storyboard.BGMVolume <= 0 {
			p.Storyboard.BGMVolume = 0.22
		}
		e.noteStudio(p, "bgm", label)
		return nil
	})
}

// ClearStudioBGM 清掉工程上的配乐编号，并记下操作者不要配乐，一键成片不会再自动配上。不删目录里的文件。
// id 传给 mutateStudio。
func (e *Engine) ClearStudioBGM(id string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		p.BGMHash = ""
		p.Storyboard.BGM = ""
		p.NoBGM = true
		p.BGMName = ""
		e.noteStudio(p, "bgm", "cleared")
		return nil
	})
}

// studioBGMPrompt 按工程的语气、风格和标题写一句配乐描述：纯音乐、不抢人声、适合短视频旁白。
func studioBGMPrompt(p StudioProject) string {
	mood := map[string]string{
		"energetic":    "节奏明快、有活力",
		"professional": "干净克制、偏商务",
		"storytelling": "温暖舒缓、有叙事感",
	}[p.Tone]
	if mood == "" {
		mood = "轻松自然"
	}
	parts := []string{mood + "的纯音乐垫乐", "没有人声", "适合短视频旁白下面铺底", "时长约 60 秒，可以循环"}
	if s := strings.TrimSpace(p.Style); s != "" {
		parts = append(parts, "画面风格："+s)
	}
	if t := strings.TrimSpace(p.Title); t != "" {
		parts = append(parts, "主题："+t)
	}
	return strings.Join(parts, "，")
}

// GenerateStudioBGM 用已启用的配乐连接生成一段配乐，写上工程，同时存进配乐目录，以后可以直接选。
// prompt 空着按工程的语气、风格和标题写一句。没有配乐连接、连接的协议还不能直接请求时返回能照着做的错误。
func (e *Engine) GenerateStudioBGM(id, prompt string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	pvd, err := e.ActiveProvider("music", "")
	if err != nil {
		return p, fmt.Errorf("还没有配乐连接。到 设置 → 生成 里启用 MiniMax、Mureka、ElevenLabs 或 Stability 的配乐，或把 mp3 放进 %s", e.bgmDir())
	}
	key, err := e.leaseProvider(pvd)
	if err != nil {
		return p, err
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = studioBGMPrompt(p)
	}
	raw, err := e.scoreMusic(e.speechCtx(), pvd, e.protocolOf(pvd, "music"), key, prompt, "")
	if err != nil {
		return p, err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	ext := audioFormat(raw)
	if ext == "" {
		ext = "mp3"
	}
	name := fmt.Sprintf("ai-%s-%s.%s", time.Now().Format("20060102-1504"), hash[:6], ext)
	if err := os.MkdirAll(e.bgmDir(), 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(e.bgmDir(), name), raw, 0o644)
	}
	return e.setStudioBGM(id, hash, "AI 配乐 · "+name)
}

// autoStudioBGM 给一键成片配上配乐：操作者说过不要、或已经有配乐就不动；配乐目录里有曲子就随机挑一首；
// 目录是空的但启用了配乐连接就生成一段；都没有时留一条说明，告诉操作者怎么加。配乐失败不挡成片。
func (e *Engine) autoStudioBGM(id string) {
	p, err := e.GetStudioProject(id)
	if err != nil || p.NoBGM || p.BGMHash != "" {
		return
	}
	if len(e.ListStudioBGM().Items) > 0 {
		if _, err := e.AttachStudioBGM(id, "random"); err == nil {
			return
		}
	}
	if _, err := e.ActiveProvider("music", ""); err == nil {
		if _, err := e.GenerateStudioBGM(id, ""); err == nil {
			return
		} else {
			_, _ = e.mutateStudio(id, func(p *StudioProject) error {
				e.warnStudio(p, GateCompose, "配乐没生成出来，这条先不带配乐："+err.Error())
				return nil
			})
			return
		}
	}
	_, _ = e.mutateStudio(id, func(p *StudioProject) error {
		e.noteStudio(p, "bgm", "配乐目录是空的，这条没有配乐。把 mp3 放进 "+e.bgmDir()+"，或启用配乐连接，以后会自动配上。")
		return nil
	})
}
