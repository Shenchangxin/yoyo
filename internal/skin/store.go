package skin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	Root string
}

func NewStore(root string) *Store {
	return &Store{Root: root}
}

func (s *Store) dir(id string) (string, error) {
	if !ValidUserID(id) {
		return "", fmt.Errorf("invalid skin id")
	}
	return filepath.Join(s.Root, id), nil
}

func (s *Store) List() []Info {
	out := []Info{}
	ents, err := os.ReadDir(s.Root)
	if err != nil {
		return out
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		p, err := s.Load(e.Name())
		if err != nil {
			continue
		}
		out = append(out, p.Info())
	}
	return out
}

func (s *Store) Load(id string) (Pack, error) {
	dir, err := s.dir(id)
	if err != nil {
		return Pack{}, err
	}
	rawMan, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Pack{}, err
	}
	var man Manifest
	if err := json.Unmarshal(rawMan, &man); err != nil {
		return Pack{}, err
	}
	if err := man.Normalize(); err != nil {
		return Pack{}, err
	}
	man.ID = id
	dark, err := readTokens(filepath.Join(dir, "tokens", "dark.json"))
	if err != nil {
		return Pack{}, err
	}
	light, err := readTokens(filepath.Join(dir, "tokens", "light.json"))
	if err != nil {
		return Pack{}, err
	}
	files := map[string][]byte{"manifest.json": rawMan}
	if b, err := os.ReadFile(filepath.Join(dir, "tokens", "dark.json")); err == nil {
		files["tokens/dark.json"] = b
	}
	if b, err := os.ReadFile(filepath.Join(dir, "tokens", "light.json")); err == nil {
		files["tokens/light.json"] = b
	}
	add := func(rel string) {
		if rel == "" {
			return
		}
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if b, err := os.ReadFile(p); err == nil {
			files[rel] = b
		}
	}
	if man.Preview != "" {
		add(man.Preview)
	}
	if man.Wallpaper != "" {
		add("assets/" + man.Wallpaper)
	}
	if man.FontSans != "" {
		add("assets/fonts/" + man.FontSans)
	}
	if man.FontMono != "" {
		add("assets/fonts/" + man.FontMono)
	}
	return Pack{Manifest: man, Dark: dark, Light: light, Files: files}, nil
}

func readTokens(path string) (TokenMap, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TokenMap{}, nil
		}
		return nil, err
	}
	return parseTokens(b)
}

func (s *Store) Save(p Pack) (Pack, error) {
	if err := p.Manifest.Normalize(); err != nil {
		return Pack{}, err
	}
	if p.Manifest.ID == "" || !ValidUserID(p.Manifest.ID) {
		p.Manifest.ID = NewUserID()
	}
	if err := ValidateMap(p.Dark); err != nil {
		return Pack{}, fmt.Errorf("dark: %w", err)
	}
	if err := ValidateMap(p.Light); err != nil {
		return Pack{}, fmt.Errorf("light: %w", err)
	}
	if err := p.checkContrast(); err != nil {
		return Pack{}, err
	}
	dir, err := s.dir(p.Manifest.ID)
	if err != nil {
		return Pack{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "tokens"), 0o755); err != nil {
		return Pack{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets", "fonts"), 0o755); err != nil {
		return Pack{}, err
	}
	manBytes, err := json.MarshalIndent(p.Manifest, "", "  ")
	if err != nil {
		return Pack{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manBytes, 0o644); err != nil {
		return Pack{}, err
	}
	dark, _ := json.MarshalIndent(p.Dark, "", "  ")
	light, _ := json.MarshalIndent(p.Light, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "tokens", "dark.json"), dark, 0o644); err != nil {
		return Pack{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "tokens", "light.json"), light, 0o644); err != nil {
		return Pack{}, err
	}
	writeRel := func(rel string, body []byte) error {
		if rel == "" || len(body) == 0 {
			return nil
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, body, 0o644)
	}
	if p.Manifest.Preview != "" {
		_ = writeRel(p.Manifest.Preview, p.Files[p.Manifest.Preview])
	}
	if p.Manifest.Wallpaper != "" {
		key := "assets/" + p.Manifest.Wallpaper
		_ = writeRel(key, p.Files[key])
	}
	if p.Manifest.FontSans != "" {
		key := "assets/fonts/" + p.Manifest.FontSans
		_ = writeRel(key, p.Files[key])
	}
	if p.Manifest.FontMono != "" {
		key := "assets/fonts/" + p.Manifest.FontMono
		_ = writeRel(key, p.Files[key])
	}
	return s.Load(p.Manifest.ID)
}

func (s *Store) Delete(id string) error {
	dir, err := s.dir(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func (s *Store) Export(id string) ([]byte, error) {
	p, err := s.Load(id)
	if err != nil {
		return nil, err
	}
	return p.Zip()
}

func (p Pack) Zip() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, body []byte) error {
		if len(body) == 0 {
			return nil
		}
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	}
	man, _ := json.MarshalIndent(p.Manifest, "", "  ")
	if err := add("manifest.json", man); err != nil {
		return nil, err
	}
	dark, _ := json.MarshalIndent(p.Dark, "", "  ")
	light, _ := json.MarshalIndent(p.Light, "", "  ")
	if err := add("tokens/dark.json", dark); err != nil {
		return nil, err
	}
	if err := add("tokens/light.json", light); err != nil {
		return nil, err
	}
	for name, body := range p.Files {
		if name == "manifest.json" || name == "tokens/dark.json" || name == "tokens/light.json" {
			continue
		}
		if err := add(name, body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Store) File(id, rel string) (string, []byte, error) {
	if !ValidUserID(id) {
		return "", nil, fmt.Errorf("invalid skin id")
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || strings.Contains(rel, "..") {
		return "", nil, fmt.Errorf("illegal path")
	}
	if _, ok := allowedFiles[rel]; !ok {
		return "", nil, fmt.Errorf("not a skin asset")
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", nil, err
	}
	full := filepath.Join(root, id, filepath.FromSlash(rel))
	full, err = filepath.Abs(full)
	if err != nil {
		return "", nil, err
	}
	if !strings.HasPrefix(full, root+string(os.PathSeparator)) && full != root {
		return "", nil, fmt.Errorf("path escapes skin store")
	}
	if resolved, err := filepath.EvalSymlinks(full); err == nil {
		full = resolved
		rootResolved, err := filepath.EvalSymlinks(root)
		if err == nil {
			root = rootResolved
		}
		if !strings.HasPrefix(full, root+string(os.PathSeparator)) {
			return "", nil, fmt.Errorf("path escapes skin store")
		}
	}
	st, err := os.Stat(full)
	if err != nil || st.IsDir() {
		return "", nil, fmt.Errorf("not found")
	}
	body, err := os.ReadFile(full)
	if err != nil {
		return "", nil, err
	}
	return mimeFor(rel), body, nil
}

func mimeFor(rel string) string {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}
