// 剧本条目的合并和按预算渲染。不把剧本写入对象库。演化合并条目，运行时按预算渲染。

package artifact

import (
	"sort"
	"strings"
)

// ApplyDelta 在原有条目上合并增量，返回新剧本，不改接收者。removeIDs 里的编号会丢掉，丢掉的正文不占位，同文案可以再加回来。比较时忽略大小写和两端空白，留下的是先出现的那份原文，Helpful 和 Harmful 相加。
func (p Playbook) ApplyDelta(add []PlaybookBullet, removeIDs []string) Playbook {
	drop := map[string]bool{}
	for _, id := range removeIDs {
		drop[id] = true
	}
	seen := map[string]int{}
	out := Playbook{ID: p.ID}
	for _, b := range p.Bullets {
		if drop[b.ID] {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(b.Text))
		if i, ok := seen[key]; ok {
			out.Bullets[i].Helpful += b.Helpful
			out.Bullets[i].Harmful += b.Harmful
			continue
		}
		seen[key] = len(out.Bullets)
		out.Bullets = append(out.Bullets, b)
	}
	for _, b := range add {
		if drop[b.ID] {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(b.Text))
		if i, ok := seen[key]; ok {
			out.Bullets[i].Helpful += b.Helpful
			out.Bullets[i].Harmful += b.Harmful
			continue
		}
		seen[key] = len(out.Bullets)
		out.Bullets = append(out.Bullets, b)
	}
	return out
}

// Render 把条数上限乘 80 当成字节预算交给 RenderBudget。结果小于等于 0 时，那边按 2000 字节计。
func (p Playbook) Render(max int) string {
	return p.RenderBudget(max * 80)
}

// RenderBudget 按 Helpful 减 Harmful 从高到低挑选，用字节数卡预算，连同「- 」前缀和换行一起计入。Harmful 大于 Helpful+2 的条目不出现，也不改写条目正文。预算小于等于 0 时按 2000。分数相同则 ID 较小的在前。已经写出正文之后，再加一行会超预算就停；第一条即使超预算也会写入。
func (p Playbook) RenderBudget(maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 2000
	}
	type scored struct {
		b     PlaybookBullet
		score int
	}
	var items []scored
	for _, bullet := range p.Bullets {
		if bullet.Harmful > bullet.Helpful+2 {
			continue
		}
		items = append(items, scored{b: bullet, score: bullet.Helpful - bullet.Harmful})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].b.ID < items[j].b.ID
	})
	var b strings.Builder
	used := 0
	for _, it := range items {
		line := "- " + it.b.Text + "\n"
		if used+len(line) > maxRunes && used > 0 {
			break
		}
		b.WriteString(line)
		used += len(line)
	}
	return b.String()
}
