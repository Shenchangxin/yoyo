// 本文件拥有交给外部工人的技能清单：SkillRef 和说明里的技能目录段。
// 不拥有技能的发现和安装，那些在 app 和 runtime。外部 agent 没有 yoyo 的技能工具，只按路径自己读 SKILL.md。

package superagent

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SkillRef 是交给外部工人的一条技能：名字、一句说明和 SKILL.md 所在目录。
type SkillRef struct {
	Name, Description, Dir string
}

// skillCatalogRunes 是外部工人说明里每条技能说明的长度上限。
const skillCatalogRunes = 120

// skillCatalogMax 是外部工人说明里最多列出几条技能。
const skillCatalogMax = 40

// writeSkillCatalog 写出操作者装好的技能：名字、一句说明和 SKILL.md 路径。没有名字或目录的条目跳过，没有可列项时什么都不写。
func writeSkillCatalog(b *strings.Builder, skills []SkillRef) {
	var rows []string
	for _, s := range skills {
		name := strings.TrimSpace(s.Name)
		dir := strings.TrimSpace(s.Dir)
		if name == "" || dir == "" {
			continue
		}
		desc := strings.Join(strings.Fields(s.Description), " ")
		if r := []rune(desc); len(r) > skillCatalogRunes {
			desc = string(r[:skillCatalogRunes]) + "…"
		}
		path := filepath.Join(dir, "SKILL.md")
		if desc != "" {
			rows = append(rows, fmt.Sprintf("- %s：%s（%s）", name, desc, path))
		} else {
			rows = append(rows, fmt.Sprintf("- %s（%s）", name, path))
		}
		if len(rows) >= skillCatalogMax {
			break
		}
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("操作者在 yoyo 里装好的技能如下，目录只读。任务和某个技能对得上时，先读它的 SKILL.md 再照做；说明里写了 @skill:名字 时必须先读那个技能。技能里提到的其他文件在同一目录下按相对路径读。\n")
	for _, row := range rows {
		b.WriteString(row)
		b.WriteByte('\n')
	}
}
