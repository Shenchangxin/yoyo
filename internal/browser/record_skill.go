// 本文件把录下的步骤写成一份技能草稿。正文里的工具名和英文句子是生成给技能文件的数据，不在注释里改写。
// 草稿要求先做 interactive 快照，并说明不要记下密码、令牌、一次性验证码和 cookie。

package browser

import (
	"fmt"
	"strings"
)

// generateRecordedSkill 按步骤写成带 front matter 的 Markdown。没有技能名时用 recorded-browser-workflow。秘密类输入不写示例。
func generateRecordedSkill(doc StepsDocument) string {
	name := doc.SkillName
	if name == "" {
		name = "recorded-browser-workflow"
	}
	title := strings.ReplaceAll(name, "-", " ")
	var b strings.Builder
	fmt.Fprintf(&b, "---\nname: %s\n", name)
	fmt.Fprintf(&b, "description: Replay the recorded browser workflow from %s.\n", emptyDash(doc.StartURL, "the recorded start page"))
	b.WriteString("allowed-tools: browser_open browser_snapshot browser_click browser_fill browser_select browser_press browser_scroll browser_act browser_tabs browser_takeover load_skill\n")
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s\n\n", title)
	b.WriteString("## When to use\n\n")
	b.WriteString("Use this skill when the operator asks to repeat the recorded browser workflow. Provide the required inputs.\n\n")
	b.WriteString("## Required inputs\n\n")
	if len(doc.Inputs) == 0 {
		b.WriteString("None.\n\n")
	} else {
		for _, in := range doc.Inputs {
			fmt.Fprintf(&b, "- `%s` (%s)", in.Name, in.Kind)
			if in.Example != "" && in.Kind != "secret" {
				fmt.Fprintf(&b, " example `%s`", in.Example)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Workflow\n\n")
	b.WriteString("1. `browser_snapshot` `level=interactive` before every click or fill. Refs expire on navigation.\n")
	if doc.StartURL != "" {
		fmt.Fprintf(&b, "2. `browser_open` `%s` if the isolated browser is not already there.\n", doc.StartURL)
	}
	n := 3
	if doc.StartURL == "" {
		n = 2
	}
	for _, st := range doc.Steps {
		if st.Kind == "open" {
			continue
		}
		fmt.Fprintf(&b, "%d. %s\n", n, describeSkillStep(st))
		n++
	}
	b.WriteString("\n## Target matching\n\n")
	b.WriteString("Prefer `data-testid`, then role + accessible name, then label/placeholder, then a stable CSS selector. Never hardcode coordinates.\n")
	b.WriteString("Do not log passwords, tokens, one-time codes, or cookies. Keep secrets as `{{sensitive_value}}` and ask the operator.\n")
	return b.String()
}

// describeSkillStep 把一步写成草稿里的一句操作说明。不认识的 kind 用步骤自带的 Description。
func describeSkillStep(st SemanticStep) string {
	name := targetName(st.Target)
	switch st.Kind {
	case "navigate":
		return "`browser_open` `" + st.URL + "`"
	case "click":
		return "Find **" + name + "** on the interactive snapshot and `browser_click` its `ref=`"
	case "fill":
		return "Fill **" + name + "** with `" + st.Value + "` using `browser_fill` (replace, do not append)"
	case "select":
		return "`browser_select` **" + name + "** to `" + st.Value + "`"
	case "press":
		return "`browser_press` `" + st.Key + "`"
	case "scroll":
		return "`browser_scroll` down so the next target is in view"
	default:
		return st.Description
	}
}

// emptyDash 在 s 空白时返回 fallback。
func emptyDash(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
