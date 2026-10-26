// Harbor 之前的确定性准入。多种 L1 载荷、tool_use、held 编号、仓库概览、照抄 held-in 说明、剧本超长和非行动技能都会拒绝。
// AdmitOverlay 在此之上限制 INV 与 DOM 的表面。不跑评测。

package evolve

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// payloadCount 数提议上非空的 L1 载荷种数。
func payloadCount(p Proposal) int {
	n := 0
	if p.Fragment != nil {
		n++
	}
	if p.PlaybookBullet != nil {
		n++
	}
	if p.SkillMD != "" {
		n++
	}
	if p.InstructionText != "" {
		n++
	}
	if p.MiddlewareText != "" {
		n++
	}
	return n
}

// proposalText 把载荷、Expected 和 Audit 拼成一段，供检索 held 编号和概览用语。
func proposalText(p Proposal) string {
	var b strings.Builder
	if p.Fragment != nil {
		b.WriteString(p.Fragment.Text)
		b.WriteByte('\n')
	}
	if p.PlaybookBullet != nil {
		b.WriteString(p.PlaybookBullet.Text)
		b.WriteByte('\n')
	}
	b.WriteString(p.SkillMD)
	b.WriteString(p.InstructionText)
	b.WriteString(p.MiddlewareText)
	b.WriteString(p.Expected)
	b.WriteString(p.Audit)
	return b.String()
}

// surfaceKey 优先用 Surface。为空时按载荷种类回落，都没有时为 unknown。
// p 的 Surface 非空时直接返回；否则按 Fragment、PlaybookBullet、SkillMD、InstructionText、MiddlewareText 回落，都没有时返回 unknown。
func surfaceKey(p Proposal) string {
	if p.Surface != "" {
		return p.Surface
	}
	switch {
	case p.Fragment != nil:
		return "prompt_fragment"
	case p.PlaybookBullet != nil:
		return "playbook"
	case p.SkillMD != "":
		return "skill"
	case p.InstructionText != "":
		return "instruction"
	case p.MiddlewareText != "":
		return "middleware"
	default:
		return "unknown"
	}
}

// ModuleOf 把 surface 收成 observation、completion、agent_loop 或 context。middleware 是 observation。verification 槽的 instruction 是 completion，其他 instruction 是 agent_loop。playbook、prompt_fragment 和 skill 是 context。
func ModuleOf(p Proposal) string {
	switch p.Surface {
	case "middleware":
		return "observation"
	case "instruction":
		if p.InstructionSlot == "verification" {
			return "completion"
		}
		return "agent_loop"
	case "playbook", "prompt_fragment", "skill":
		return "context"
	default:
		if p.MiddlewareText != "" {
			return "observation"
		}
		if p.InstructionText != "" && p.InstructionSlot == "verification" {
			return "completion"
		}
		return "context"
	}
}

// AdmitQuality 在花评测之前拒绝提议。载荷多于一种、surface 为 tool_use、正文含 held 编号、像仓库概览、照抄 held-in 说明的前 80 个字节（说明折叠空白后至少 40 字节）、剧本 rune 数超过上限，或技能被 fluffSkill 判为无效时返回错误。
// playbookTokens 小于等于 0 时按 2000 计。
// pb 只在有 PlaybookBullet 时用来 ApplyDelta，再交给 renderPlaybook 数 rune。
// heldInText 的每一条交给 leakedInstruction，命中则返回 copied held-in instruction。
func AdmitQuality(p Proposal, pb artifact.Playbook, playbookTokens int, held map[string]bool, heldInText []string) error {
	if payloadCount(p) > 1 {
		return fmt.Errorf("multiple L1 surfaces")
	}
	if p.Surface == "tool_use" {
		return fmt.Errorf("tool_use is frozen")
	}
	text := proposalText(p)
	lower := strings.ToLower(text)
	for id := range held {
		if id != "" && strings.Contains(text, id) {
			return fmt.Errorf("held-out id in proposal")
		}
	}
	if looksLikeRepoOverview(lower) {
		return fmt.Errorf("repo-overview dump")
	}
	for _, inst := range heldInText {
		if leakedInstruction(lower, inst) {
			return fmt.Errorf("copied held-in instruction")
		}
	}
	if p.PlaybookBullet != nil {
		next := pb.ApplyDelta([]artifact.PlaybookBullet{*p.PlaybookBullet}, nil)
		if playbookTokens <= 0 {
			playbookTokens = 2000
		}
		if utf8.RuneCountInString(renderPlaybook(next)) > playbookTokens {
			return fmt.Errorf("playbook token cap")
		}
	}
	if p.SkillMD != "" && fluffSkill(p.SkillMD) {
		return fmt.Errorf("non-actionable skill")
	}
	return nil
}

// AdmitOverlay 先走 AdmitQuality。INV 拒绝剧本和常开片段；instruction 的槽只许 verification 或 failure_recovery，空槽按 verification，正文不能为空，rune 数不能超过 overlayAlwaysOnMax，也不能带触发语。middleware 和 skill 的正文不能为空。
// DOM 只许 surface 为 skill 且 SkillMD 非空，不能再带剧本、片段、指令或中间件。其他维度返回 unknown overlay dim。
// p 先交给 AdmitQuality；维度为 INV 时再看 PlaybookBullet、Surface、Fragment、InstructionSlot、InstructionText、MiddlewareText、SkillMD，为 DOM 时要求 Surface 是 skill 且 SkillMD 非空。
// pb 交给 AdmitQuality。
// playbookTokens 交给 AdmitQuality。
// held 交给 AdmitQuality。
// heldInText 交给 AdmitQuality。
func AdmitOverlay(p Proposal, dim string, pb artifact.Playbook, playbookTokens int, held map[string]bool, heldInText []string) error {
	if err := AdmitQuality(p, pb, playbookTokens, held, heldInText); err != nil {
		return err
	}
	switch strings.ToUpper(strings.TrimSpace(dim)) {
	case "INV":
		if p.PlaybookBullet != nil || p.Surface == "playbook" {
			return fmt.Errorf("inv playbook")
		}
		if p.Fragment != nil || p.Surface == "prompt_fragment" {
			return fmt.Errorf("inv always-on fragment")
		}
		switch p.Surface {
		case "instruction":
			slot := p.InstructionSlot
			if slot == "" {
				slot = "verification"
			}
			if slot != "verification" && slot != "failure_recovery" {
				return fmt.Errorf("inv instruction slot")
			}
			if p.InstructionText == "" {
				return fmt.Errorf("inv empty instruction")
			}
			if utf8.RuneCountInString(p.InstructionText) > overlayAlwaysOnMax {
				return fmt.Errorf("always-on cap")
			}
			if instructionHasTrigger(p.InstructionText) {
				return fmt.Errorf("always-on with trigger")
			}
		case "middleware":
			if p.MiddlewareText == "" {
				return fmt.Errorf("inv empty middleware")
			}
		case "skill":
			if p.SkillMD == "" {
				return fmt.Errorf("inv empty skill")
			}
		default:
			return fmt.Errorf("inv surface")
		}
	case "DOM":
		if p.Surface != "skill" || p.SkillMD == "" {
			return fmt.Errorf("dom must be skill")
		}
		if p.PlaybookBullet != nil || p.Fragment != nil || p.InstructionText != "" || p.MiddlewareText != "" {
			return fmt.Errorf("dom extra surface")
		}
	default:
		return fmt.Errorf("unknown overlay dim")
	}
	return nil
}

// instructionHasTrigger 在正文里找 when、whenever、if the user、if a user、after the user，以及「当用户」「当请求」「如果用户」「若用户」。
// text 转成小写后查找这些英文片段；原文再查找那四句中文。
func instructionHasTrigger(text string) bool {
	lower := strings.ToLower(text)
	for _, p := range []string{
		"when ", "whenever ", "if the user", "if a user", "after the user",
		"当用户", "当请求", "如果用户", "若用户",
	} {
		if strings.Contains(lower, strings.ToLower(p)) || strings.Contains(text, p) {
			return true
		}
	}
	return false
}

// looksLikeRepoOverview 在小写正文里找仓库概览、目录结构、this repo contains、agents.md 和 warehouse tour。
// lower 里查找 repository overview、directory structure、this repo contains、agents.md、warehouse tour。
func looksLikeRepoOverview(lower string) bool {
	for _, n := range []string{"repository overview", "directory structure", "this repo contains", "agents.md", "warehouse tour"} {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

// leakedInstruction 把说明折成单行小写。短于 40 字节则不算泄漏。否则取前 80 字节，看提议小写正文是否包含这段。
// lower 用来查找截断后的说明。
// inst 先折成单行小写；短于 40 字节返回 false；长于 80 字节只取前 80。
func leakedInstruction(lower, inst string) bool {
	inst = strings.ToLower(strings.Join(strings.Fields(inst), " "))
	if len(inst) < 40 {
		return false
	}
	clip := inst
	if len(clip) > 80 {
		clip = clip[:80]
	}
	return strings.Contains(lower, clip)
}

// fluffSkill 去掉前置信息后，正文 rune 数小于 1200 则不算空话。达到 1200 时，write、run、check、verify、call、read 里命中少于 2 个则算空话。
// md 以 --- 开头时去掉到下一处 --- 为止的前置信息，再按正文 rune 数和那六个词判断。
func fluffSkill(md string) bool {
	body := md
	if strings.HasPrefix(md, "---") {
		rest := md[3:]
		if i := strings.Index(rest, "---"); i >= 0 {
			body = rest[i+3:]
		}
	}
	if utf8.RuneCountInString(body) < 1200 {
		return false
	}
	lower := strings.ToLower(body)
	hits := 0
	for _, w := range []string{"write", "run", "check", "verify", "call", "read"} {
		if strings.Contains(lower, w) {
			hits++
		}
	}
	return hits < 2
}

// renderPlaybook 把每条剧本正文按行接起来，供 rune 计数。
// pb 的每条 Bullet 正文各写一行。
func renderPlaybook(pb artifact.Playbook) string {
	var b strings.Builder
	for _, x := range pb.Bullets {
		b.WriteString(x.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
