package evolve

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

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

// AdmitQuality is a deterministic pre-Harbor gate. Failures must not spend eval.
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

// AdmitOverlay is AdmitQuality plus the session-PPC surface contract.
// INV may write instruction (verification|failure_recovery), middleware, or a
// procedure skill. DOM may write only an on-demand skill. Playbook is global
// ACE memory and is never an overlay surface.
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

func looksLikeRepoOverview(lower string) bool {
	for _, n := range []string{"repository overview", "directory structure", "this repo contains", "agents.md", "warehouse tour"} {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

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

func renderPlaybook(pb artifact.Playbook) string {
	var b strings.Builder
	for _, x := range pb.Bullets {
		b.WriteString(x.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
