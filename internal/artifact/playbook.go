package artifact

import "strings"

// ApplyDelta merges ACE-style incremental bullets without rewriting the book.
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

func (p Playbook) Render(max int) string {
	return p.RenderBudget(max * 80)
}

// RenderBudget keeps high-value ACE bullets under a token-ish rune budget.
// Harmful-heavy bullets stay out (grow-and-refine), never a full rewrite.
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
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
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
