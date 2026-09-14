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
	if max <= 0 {
		max = 32
	}
	var b strings.Builder
	n := 0
	for _, bullet := range p.Bullets {
		if bullet.Harmful > bullet.Helpful+2 {
			continue
		}
		b.WriteString("- ")
		b.WriteString(bullet.Text)
		b.WriteByte('\n')
		n++
		if n >= max {
			break
		}
	}
	return b.String()
}
