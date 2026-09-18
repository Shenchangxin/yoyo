package expert

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
)

// Pack is a Harbor-admissible expert: skill + playbook bullets + eval ids + policy.
type Pack struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SkillMD     string   `json:"skill_md"`
	Playbook    []string `json:"playbook,omitempty"`
	EvalIDs     []string `json:"eval_ids,omitempty"`
	Policy      string   `json:"policy,omitempty"`
	SigHex      string   `json:"sig_hex,omitempty"`
}

func ParseDir(dir string) (Pack, error) {
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return Pack{}, err
	}
	sk, err := artifact.ParseSkillMD(string(b), dir)
	if err != nil {
		return Pack{}, err
	}
	p := Pack{Name: sk.Name, Description: sk.Description, SkillMD: string(b)}
	if pb, err := os.ReadFile(filepath.Join(dir, "PLAYBOOK.txt")); err == nil {
		for _, line := range strings.Split(string(pb), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				p.Playbook = append(p.Playbook, line)
			}
		}
	}
	if pol, err := os.ReadFile(filepath.Join(dir, "POLICY.md")); err == nil {
		p.Policy = string(pol)
	}
	if ev, err := os.ReadFile(filepath.Join(dir, "EVALS.txt")); err == nil {
		for _, line := range strings.Split(string(ev), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				p.EvalIDs = append(p.EvalIDs, line)
			}
		}
	}
	if sig, err := os.ReadFile(filepath.Join(dir, "SKILL.sig")); err == nil {
		p.SigHex = strings.TrimSpace(string(sig))
	}
	return p, nil
}

func Verify(pub ed25519.PublicKey, p Pack) error {
	if len(pub) == 0 {
		if p.SigHex == "" {
			return fmt.Errorf("expert: unsigned pack %s cannot become refs/active", p.Name)
		}
		return fmt.Errorf("expert: missing public key")
	}
	sig, err := wasm.ParseSig(p.SigHex)
	if err != nil {
		return err
	}
	return wasm.Verify(pub, []byte(p.SkillMD), sig)
}

func Sign(priv ed25519.PrivateKey, skillMD string) string {
	return hex.EncodeToString(wasm.Sign(priv, []byte(skillMD)))
}

func Admit(dir, destRoot string, pub ed25519.PublicKey) (Pack, error) {
	p, err := ParseDir(dir)
	if err != nil {
		return Pack{}, err
	}
	if err := Verify(pub, p); err != nil {
		return Pack{}, err
	}
	if destRoot == "" {
		return p, nil
	}
	out := filepath.Join(destRoot, p.Name)
	if err := os.MkdirAll(out, 0o755); err != nil {
		return Pack{}, err
	}
	if err := os.WriteFile(filepath.Join(out, "SKILL.md"), []byte(p.SkillMD), 0o644); err != nil {
		return Pack{}, err
	}
	if p.SigHex != "" {
		_ = os.WriteFile(filepath.Join(out, "SKILL.sig"), []byte(p.SigHex+"\n"), 0o644)
	}
	if len(p.EvalIDs) > 0 {
		_ = os.WriteFile(filepath.Join(out, "EVALS.txt"), []byte(strings.Join(p.EvalIDs, "\n")+"\n"), 0o644)
	}
	if len(p.Playbook) > 0 {
		_ = os.WriteFile(filepath.Join(out, "PLAYBOOK.txt"), []byte(strings.Join(p.Playbook, "\n")+"\n"), 0o644)
	}
	if p.Policy != "" {
		_ = os.WriteFile(filepath.Join(out, "POLICY.md"), []byte(p.Policy), 0o644)
	}
	return p, nil
}

func Distill(name, description, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", name)
	fmt.Fprintf(&b, "description: %s\n", description)
	b.WriteString("license: Apache-2.0\n")
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteByte('\n')
	return b.String()
}
