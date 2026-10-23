package video

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type StudioBGMItem struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}

type StudioBGMList struct {
	Dir   string          `json:"dir"`
	Items []StudioBGMItem `json:"items"`
}

func (e *Engine) bgmDir() string {
	return filepath.Join(e.Dir, "bgm")
}

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

func bgmExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".wav", ".aac", ".flac", ".ogg":
		return true
	default:
		return false
	}
}

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
	p.BGMHash = hash
	p.Storyboard.BGM = hash
	if p.Storyboard.BGMVolume <= 0 {
		p.Storyboard.BGMVolume = 0.22
	}
	e.noteStudio(&p, "bgm", name)
	return e.saveStudio(p)
}

func (e *Engine) ClearStudioBGM(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	p.BGMHash = ""
	p.Storyboard.BGM = ""
	e.noteStudio(&p, "bgm", "cleared")
	return e.saveStudio(p)
}
