// 按 tests/expect.toml 检查工作区文件是否包含指定子串，比较时忽略大小写。
// runVerifier 在个人验证器未接手、且该文件存在时调用。不执行测试脚本。

package eval

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// expectFile 是 expect.toml 里的一条：路径和应包含的子串。
type expectFile struct {
	Path     string `toml:"path"`
	Contains string `toml:"contains"`
}

// expectDoc 是 expect.toml 的 file 数组。
type expectDoc struct {
	File []expectFile `toml:"file"`
}

// runExpect 读取 toml。file 为空、文件缺失或内容对不上时返回假和 fmtNoExpect。相对路径相对 work。Contains 为空时只要求文件能读到。
// path 用 os.ReadFile 读出，再按 toml 解开。
func runExpect(work, path string) (bool, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, "", err
	}
	var doc expectDoc
	if err := toml.Unmarshal(b, &doc); err != nil {
		return false, "", err
	}
	if len(doc.File) == 0 {
		return false, "", fmtNoExpect
	}
	var missing []string
	for _, f := range doc.File {
		p := f.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(work, p)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			missing = append(missing, f.Path)
			continue
		}
		if f.Contains != "" && !strings.Contains(strings.ToLower(string(raw)), strings.ToLower(f.Contains)) {
			missing = append(missing, f.Path+" (content)")
		}
	}
	if len(missing) > 0 {
		return false, "missing " + strings.Join(missing, ", "), fmtNoExpect
	}
	return true, "ok", nil
}

// fmtNoExpect 是 expect 未通过时共用的错误。缺文件和内容不符都返回它。
var fmtNoExpect = errString("expect failed")

// errString 是一段当作 error 的文本。
type errString string

// Error 返回这段文本本身。
func (e errString) Error() string { return string(e) }
