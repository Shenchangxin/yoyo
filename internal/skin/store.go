// Package skin 在 root 下按用户 id 存放皮肤包。NewStore 不创建目录。桌面应用把 root 设成主目录的 skins/。用户 id 不能以 builtin. 开头。
// 令牌名必须在 Specs 里。Admit 只接受允许列表中的 zip 条目。File 拒绝含 .. 的路径；EvalSymlinks 成功时再确认解析结果仍在 root 下。
// PreserveEvidence 为空时 EvidenceLocked 为真，Compile 用内置调色板的证据色盖过用户值。
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

// Store 的 Root 是皮肤根目录。每个用户包在 Root/<id> 下。
type Store struct {
	Root string
}

// NewStore 只记下 root，不创建目录。
func NewStore(root string) *Store {
	return &Store{Root: root}
}

// dir 在 id 通过 ValidUserID 时返回 Root/<id>，否则返回 invalid skin id。
func (s *Store) dir(id string) (string, error) {
	if !ValidUserID(id) {
		return "", fmt.Errorf("invalid skin id")
	}
	return filepath.Join(s.Root, id), nil
}

// List 返回根下每个能 Load 的子目录的 Info。读目录失败时返回空切片。
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

// Load 读取 manifest.json 以及 tokens/dark.json、tokens/light.json，并按清单带上预览、壁纸和字体字节。
// id 传给 dir，并写入清单的 ID。
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

// readTokens 读取 JSON 令牌文件。文件不存在时返回空 TokenMap 和 nil。
// path 交给 os.ReadFile。不存在时返回空 TokenMap 和 nil，否则把字节传给 parseTokens。
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

// Save 先规范化清单、校验两套令牌和对比度。id 为空或非法时换成 NewUserID。然后写入清单、令牌和列出的资源，再 Load 回来。
// p 的 Manifest 先 Normalize。ID 为空或 ValidUserID 为假时换成 NewUserID。Dark 和 Light 交给 ValidateMap，再 checkContrast。清单和两套令牌写到该 ID 的目录。
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

// Delete 在 id 合法且目录存在时删掉整个包目录。
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

// Export 先 Load，再打成 zip 字节。
// id 传给 Load。
func (s *Store) Export(id string) ([]byte, error) {
	p, err := s.Load(id)
	if err != nil {
		return nil, err
	}
	return p.Zip()
}

// Zip 把清单、两套令牌和 Files 里的其他条目打成 zip。空正文的条目跳过。
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

// File 只读 allowedFiles 里的相对路径。路径含 ..、或绝对路径不在 root 下时返回错误。EvalSymlinks 成功时，解析后的路径也必须仍在 root 下。
// id 交给 ValidUserID。为假时返回错误。通过后接在 root 下作为目录。
// rel 把反斜杠换成斜杠并去掉开头的斜杠。为空、含 ..，或不在 allowedFiles 里时返回错误。通过后传给 mimeFor。
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

// mimeFor 按扩展名返回 Content-Type。不认识的扩展名用 application/octet-stream。
// rel 的扩展名转成小写后选择类型。
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
