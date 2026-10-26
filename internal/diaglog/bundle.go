// 本文件把日志目录打成支持用的 zip。单份日志默认最多收进末尾 4 << 20 字节。
// 清单的 omits 固定列出 vault、cas、spill、memory、connectors、held_out。路径里含 vault 的文件和已有的 zip 不打进包。

package diaglog

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BundleOpts 是支持包的输入。Dest 为空时写到 Dir 下的 support-时间.zip。
type BundleOpts struct {
	Dir         string
	Dest        string
	Doctor      map[string]any
	SpansPath   string
	ConfigPath  string
	Isolation   map[string]any
	Version     string
	HeldOutIDs  []string
	MaxLogBytes int64
}

// WriteBundle 生成 zip 并返回它的路径。MaxLogBytes 小于等于 0 时用 4 << 20。
// 父目录权限 0o755。zip 本身权限 0o600。版本、doctor、隔离、配置和跨度文件按有无分别写入。
// doctor 先 stripSecrets。配置 YAML 能解析时按键脱敏，解析失败则对原文 RedactBytes。
// 跨度文件和日志树都只保留每个文件末尾的 MaxLogBytes 字节。清单含 created、omits 和 held_out_ids。
func WriteBundle(opt BundleOpts) (string, error) {
	if opt.Dest == "" {
		opt.Dest = filepath.Join(opt.Dir, "support-"+time.Now().UTC().Format("20060102T150405Z")+".zip")
	}
	if opt.MaxLogBytes <= 0 {
		opt.MaxLogBytes = 4 << 20
	}
	if err := os.MkdirAll(filepath.Dir(opt.Dest), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(opt.Dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(f)
	add := func(name string, r io.Reader) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}
	addBytes := func(name string, b []byte) error {
		return add(name, strings.NewReader(string(RedactBytes(b))))
	}

	if opt.Version != "" {
		_ = addBytes("version.txt", []byte(opt.Version+"\n"))
	}
	if opt.Doctor != nil {
		b, _ := json.MarshalIndent(stripSecrets(opt.Doctor), "", "  ")
		_ = addBytes("doctor.json", b)
	}
	if opt.Isolation != nil {
		b, _ := json.MarshalIndent(opt.Isolation, "", "  ")
		_ = addBytes("isolation.json", b)
	}
	if opt.ConfigPath != "" {
		if raw, err := os.ReadFile(opt.ConfigPath); err == nil {
			_ = addBytes("config.yaml", redactConfigYAML(raw))
		}
	}
	if opt.SpansPath != "" {
		_ = addFileCapped(zw, opt.SpansPath, "spans.jsonl", opt.MaxLogBytes)
	}
	if opt.Dir != "" {
		_ = addLogTree(zw, opt.Dir, opt.MaxLogBytes)
	}

	manifest := map[string]any{
		"created":      time.Now().UTC().Format(time.RFC3339),
		"omits":        []string{"vault", "cas", "spill", "memory", "connectors", "held_out"},
		"held_out_ids": opt.HeldOutIDs,
	}
	b, _ := json.MarshalIndent(manifest, "", "  ")
	_ = addBytes("manifest.json", b)

	if err := zw.Close(); err != nil {
		_ = f.Close()
		return opt.Dest, err
	}
	return opt.Dest, f.Close()
}

// addLogTree 把 dir 下的文件放进 logs/ 相对路径。目录错误继续返回。以 .zip 结尾或路径里含 vault 的跳过。每个文件只取末尾 capBytes 字节。
func addLogTree(zw *zip.Writer, dir string, capBytes int64) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".zip") {
			return nil
		}
		name := strings.ToLower(rel)
		if strings.Contains(name, "vault") {
			return nil
		}
		return addFileCapped(zw, path, "logs/"+rel, capBytes)
	})
}

// addFileCapped 把文件末尾最多 capBytes 字节脱敏后写进 zip。读失败时返回 nil，不中断打包。超过上限时保留切片的尾部。
func addFileCapped(zw *zip.Writer, path, name string, capBytes int64) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if int64(len(raw)) > capBytes {
		raw = raw[int64(len(raw))-capBytes:]
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(RedactBytes(raw))
	return err
}

// redactConfigYAML 把 YAML 解成映射后 stripSecrets，再编回 YAML。解析或编码失败时对原文 RedactBytes。
func redactConfigYAML(raw []byte) []byte {
	var m map[string]any
	if yaml.Unmarshal(raw, &m) != nil {
		return RedactBytes(raw)
	}
	stripSecrets(m)
	b, err := yaml.Marshal(m)
	if err != nil {
		return RedactBytes(raw)
	}
	return b
}

// stripSecrets 递归改写映射、切片和字符串。键被 LooksSecretKey 命中，或大小写等于 search_key、SearchKey 时，值改成占位文本。
// 字符串叶子走 Redact。其他类型原样返回。
func stripSecrets(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if LooksSecretKey(k) || strings.EqualFold(k, "search_key") || strings.EqualFold(k, "SearchKey") {
				t[k] = redacted
				continue
			}
			t[k] = stripSecrets(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = stripSecrets(val)
		}
		return t
	case string:
		return Redact(t)
	default:
		return v
	}
}

// ZipContains 在 zip 的成员名或成员正文里查找 needle。打开失败返回假。任一成员读失败则跳过该成员。
func ZipContains(path, needle string) bool {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer r.Close()
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		if strings.Contains(string(b), needle) || strings.Contains(f.Name, needle) {
			return true
		}
	}
	return false
}
