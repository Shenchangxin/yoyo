package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

func (e *Engine) materialRoot() string {
	if e != nil && e.SuitesRoot != "" {
		return filepath.Join(e.SuitesRoot, ".materialized")
	}
	return filepath.Join(os.TempDir(), "yoyo-eval-tasks")
}

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

func shellQuote(s string) string {
	if strings.IndexByte(s, '\'') < 0 {
		return "'" + s + "'"
	}
	return strconvQuote(s)
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
