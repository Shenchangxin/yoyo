// Package expert 读取并验收专家包：技能正文、剧本条目、评测 id 和策略。
// 公钥为空且没有签名时，Verify 拒绝该包进入 refs/active。Admit 先验签；destRoot 为空则只返回，不写目录。
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

// Pack 是一份 Harbor 可准入的专家包，含技能、剧本要点、评测 id 和策略。
type Pack struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SkillMD     string   `json:"skill_md"`
	Playbook    []string `json:"playbook,omitempty"`
	EvalIDs     []string `json:"eval_ids,omitempty"`
	Policy      string   `json:"policy,omitempty"`
	SigHex      string   `json:"sig_hex,omitempty"`
}

// ParseDir 读取 dir 里的 SKILL.md。名字和描述来自 artifact.ParseSkillMD，正文是文件原文。
// PLAYBOOK.txt 和 EVALS.txt 若存在，跳过空行和以 # 开头的行。POLICY.md 和 SKILL.sig 若存在则原样收入，签名会去掉首尾空白。
// 可选文件缺失不算错误。SKILL.md 读失败或技能解析失败时返回错误。
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

// Verify 用公钥校验技能正文的 ed25519 签名。
// 公钥长度为 0 且 SigHex 为空时，错误写明未签名的包不能进入 refs/active。公钥为空但带了签名时，返回缺少公钥。
// 否则用 wasm.ParseSig 解析 SigHex，再对 SkillMD 的字节调用 wasm.Verify。
// pub 长度为 0 且 p 的 SigHex 为空时返回未签名错误，长度为 0 但有签名时返回缺少公钥，否则传给 wasm.Verify。
// p 在 pub 长度为 0 时看 SigHex 是否为空。否则 SigHex 交给 wasm.ParseSig，SkillMD 的字节交给 wasm.Verify。
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

// Sign 对技能正文调用 wasm.Sign，再把签名编成十六进制。
// priv 传给 wasm.Sign。
// skillMD 转成字节后传给 wasm.Sign，返回值再编成十六进制。
func Sign(priv ed25519.PrivateKey, skillMD string) string {
	return hex.EncodeToString(wasm.Sign(priv, []byte(skillMD)))
}

// Admit 先 ParseDir 再 Verify。destRoot 为空时验签通过就返回，不创建目录。
// 否则写入 destRoot/名字，目录权限 0o755。SKILL.md 权限 0o644。签名、评测、剧本和策略只在对应字段非空时写出。
// dir 传给 ParseDir。
// pub 和解析出的包一起传给 Verify。
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

// Distill 拼出一份技能 Markdown。前置信息含 name、description，license 固定写成 Apache-2.0。正文先去掉首尾空白，末尾补一个换行。
// body 去掉首尾空白后写在前置信息后面，并补一个换行。
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
