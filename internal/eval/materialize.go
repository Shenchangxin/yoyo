// 磁盘上没有 instruction.md 时，按目录规格写出任务。expect 和测试脚本写在任务自己的 tests/ 里。
// 智能体工作区由 copyDir 生成，那一步会跳过 tests/。Engine.resolveTaskDir 调用这里。

package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveTaskDir 在 root/id 已有 instruction.md 时直接用它。否则按规格物化到 materialRoot，失败时仍返回 root/id。
func (e *Engine) resolveTaskDir(root, id string) string {
	p := filepath.Join(root, id)
	if _, err := os.Stat(filepath.Join(p, "instruction.md")); err == nil {
		return p
	}
	spec, ok := lookupSpec(id)
	if !ok {
		return p
	}
	dst := filepath.Join(e.materialRoot(), id)
	if err := materialize(dst, spec); err != nil {
		return p
	}
	return dst
}

// materialRoot 在 SuitesRoot 非空时用其下的 .materialized，否则用临时目录 yoyo-eval-tasks。
func (e *Engine) materialRoot() string {
	if e != nil && e.SuitesRoot != "" {
		return filepath.Join(e.SuitesRoot, ".materialized")
	}
	return filepath.Join(os.TempDir(), "yoyo-eval-tasks")
}

// materialize 写出 instruction.md、timeout_minutes 为 1 的 task.toml，以及 tests 下的 expect.toml、test.sh 和 test.ps1。
// dir 下创建 tests，并写出 instruction.md、task.toml 和 tests 里的那三个文件。
// spec 的 File 和 Contains 写进 instruction、expect.toml 和两份测试脚本。
func materialize(dir string, spec TaskSpec) error {
	if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
		return err
	}
	inst := fmt.Sprintf("Write a file named %s in the workspace containing exactly the word %s (lowercase).\n", spec.File, spec.Contains)
	if err := os.WriteFile(filepath.Join(dir, "instruction.md"), []byte(inst), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "task.toml"), []byte("timeout_minutes = 1\n"), 0o644); err != nil {
		return err
	}
	expect := fmt.Sprintf("[[file]]\npath = %q\ncontains = %q\n", spec.File, spec.Contains)
	if err := os.WriteFile(filepath.Join(dir, "tests", "expect.toml"), []byte(expect), 0o644); err != nil {
		return err
	}
	sh := fmt.Sprintf("#!/bin/sh\ngrep -q %s %s || exit 1\n", shellQuote(spec.Contains), spec.File)
	ps := fmt.Sprintf("if (-not (Test-Path '%s')) { exit 1 }\n$c = Get-Content '%s' -Raw\nif ($c -notmatch '%s') { exit 1 }\nexit 0\n", spec.File, spec.File, spec.Contains)
	if err := os.WriteFile(filepath.Join(dir, "tests", "test.sh"), []byte(sh), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "tests", "test.ps1"), []byte(ps), 0o644)
}

// shellQuote 在字符串不含单引号时用单引号包住，否则交给 strconvQuote。
func shellQuote(s string) string {
	if strings.IndexByte(s, '\'') < 0 {
		return "'" + s + "'"
	}
	return strconvQuote(s)
}

// strconvQuote 用双引号包住，并把其中的双引号写成反斜杠转义。
func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
