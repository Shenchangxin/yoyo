// 大纲的整份替换和零碎修改。留下来的知识点保住自己的 id 和证据，位置变了不算换了一个点。
// 删掉的知识点连同证据和题目一起清掉。前置关系不能成环。

package learn

import (
	"database/sql"
	"fmt"
	"strings"
)

// OutlineModule 是大纲里的一章。ID 为空表示新章，或按名字去认已有的章。
type OutlineModule struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	Objective string         `json:"objective,omitempty"`
	Points    []OutlinePoint `json:"points"`
}

// OutlinePoint 是大纲里的一个知识点。Kind 只能是 memory、procedure、concept、design。
// ID 指向已有的知识点。Ref 是这次请求里临时起的名字，只给 Prereqs 引用。
// Prereqs 里可以写 id、Ref 或知识点名称。为 nil 时已有知识点保留原来的前置，写成空数组才是清空。
type OutlinePoint struct {
	ID      string   `json:"id,omitempty"`
	Ref     string   `json:"ref,omitempty"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Prereqs []string `json:"prereqs,omitempty"`
}

// OutlineChange 是一次大纲改动的统计。RemovedMastered 是被删掉的已掌握知识点数。
type OutlineChange struct {
	Modules         int `json:"modules"`
	Added           int `json:"added"`
	Kept            int `json:"kept"`
	Removed         int `json:"removed"`
	RemovedMastered int `json:"removed_mastered"`
}

// OutlineOptions 控制替换方式。Append 为真时只在现有大纲后面追加。KeepMastered 为真时拒绝删掉已掌握的知识点。
type OutlineOptions struct {
	Append       bool
	KeepMastered bool
}

// Outline 返回路径当前的大纲，带 id 和前置关系。编辑器和零碎修改从它出发。
func (s *Store) Outline(pathID string) ([]OutlineModule, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := loadPath(s.db, pathID)
	if err != nil {
		return nil, err
	}
	return st.outline(), nil
}

// outline 把内存里的路径收成大纲。
func (st *pathState) outline() []OutlineModule {
	out := make([]OutlineModule, 0, len(st.Modules))
	at := map[string]int{}
	for _, m := range st.Modules {
		at[m.ID] = len(out)
		out = append(out, OutlineModule{ID: m.ID, Name: m.Name, Objective: m.Objective, Points: []OutlinePoint{}})
	}
	for _, p := range st.Points {
		i, ok := at[p.ModuleID]
		if !ok {
			continue
		}
		prereqs := append([]string{}, p.Prereqs...)
		out[i].Points = append(out[i].Points, OutlinePoint{ID: p.ID, Name: p.Name, Kind: p.Kind, Prereqs: prereqs})
	}
	return out
}

// fingerprint 是认同一个知识点用的键：名称不分大小写，加上类型。
func fingerprint(name, kind string) string {
	return strings.ToLower(strings.TrimSpace(name)) + "|" + kind
}

// ReplaceOutline 用 modules 替换路径的大纲。已有知识点靠 id 认；没写 id 的，名称和类型在新旧两边都唯一时也认作同一个。
// 认上的保住证据，认不上的旧知识点连同证据删掉。章节最多 30 个，每章最多 24 个知识点。
func (s *Store) ReplaceOutline(pathID string, modules []OutlineModule, opts OutlineOptions) (OutlineChange, error) {
	if s == nil {
		return OutlineChange{}, fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	var change OutlineChange
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		st, err := loadPath(tx, pathID)
		if err != nil {
			return err
		}
		incoming := normalizeOutline(modules)
		if opts.Append {
			incoming = appendOutline(st.outline(), incoming)
		}
		if len(incoming) > maxModules {
			return fmt.Errorf("an outline holds at most %d modules; merge some", maxModules)
		}
		for _, m := range incoming {
			if len(m.Points) > maxPointsPerModule {
				return fmt.Errorf("module %q has more than %d knowledge points; split it", m.Name, maxPointsPerModule)
			}
		}
		change, err = applyOutline(tx, st, incoming, opts.KeepMastered)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE paths SET updated_at = ?, revision = revision + 1 WHERE id = ?`, now, pathID)
		return err
	})
	s.mu.Unlock()
	if err != nil {
		return OutlineChange{}, err
	}
	s.notify("path", pathID, "outline")
	return change, nil
}

// normalizeOutline 去掉空名字的知识点，规范类型，截断过长的名字。没有名字的章按序号起名。
func normalizeOutline(modules []OutlineModule) []OutlineModule {
	out := make([]OutlineModule, 0, len(modules))
	for i, m := range modules {
		name := clipRunes(strings.TrimSpace(m.Name), maxNameRunes)
		if name == "" {
			name = fmt.Sprintf("Module %d", i+1)
		}
		mod := OutlineModule{ID: strings.TrimSpace(m.ID), Name: name, Objective: clipRunes(strings.TrimSpace(m.Objective), 400)}
		for _, p := range m.Points {
			pname := clipRunes(strings.TrimSpace(p.Name), maxNameRunes)
			if pname == "" {
				continue
			}
			pt := OutlinePoint{ID: strings.TrimSpace(p.ID), Ref: strings.TrimSpace(p.Ref), Name: pname, Kind: normalizeKind(p.Kind)}
			if p.Prereqs != nil {
				pt.Prereqs = []string{}
				for _, pre := range p.Prereqs {
					if pre = strings.TrimSpace(pre); pre != "" {
						pt.Prereqs = append(pt.Prereqs, pre)
					}
				}
			}
			mod.Points = append(mod.Points, pt)
		}
		out = append(out, mod)
	}
	return out
}

// appendOutline 把 extra 接在 current 后面。extra 里和现有章同名的章并进那一章。
func appendOutline(current, extra []OutlineModule) []OutlineModule {
	at := map[string]int{}
	for i, m := range current {
		at[strings.ToLower(m.Name)] = i
	}
	for _, m := range extra {
		if i, ok := at[strings.ToLower(m.Name)]; ok {
			current[i].Points = append(current[i].Points, m.Points...)
			if current[i].Objective == "" {
				current[i].Objective = m.Objective
			}
			continue
		}
		m.ID = ""
		at[strings.ToLower(m.Name)] = len(current)
		current = append(current, m)
	}
	return current
}

// applyOutline 把 incoming 写进库：认出留下来的知识点，删掉认不上的，重写顺序和前置关系。
// keepMastered 为真时，要删已掌握的知识点就报错。
func applyOutline(tx *sql.Tx, st *pathState, incoming []OutlineModule, keepMastered bool) (OutlineChange, error) {
	var change OutlineChange
	oldByID := map[string]pointRow{}
	oldPrint := map[string][]string{}
	for _, p := range st.Points {
		oldByID[p.ID] = p
		key := fingerprint(p.Name, p.Kind)
		oldPrint[key] = append(oldPrint[key], p.ID)
	}
	newPrint := map[string]int{}
	for _, m := range incoming {
		for _, p := range m.Points {
			newPrint[fingerprint(p.Name, p.Kind)]++
		}
	}
	// 先认 id，再认名称和类型。写了 id 但类型变了的，算一个新知识点。
	used := map[string]bool{}
	type slot struct{ module, point int }
	finalID := map[slot]string{}
	for mi, m := range incoming {
		for pi, p := range m.Points {
			if old, ok := oldByID[p.ID]; ok && !used[p.ID] && old.Kind == p.Kind {
				used[p.ID] = true
				finalID[slot{mi, pi}] = p.ID
			}
		}
	}
	for mi, m := range incoming {
		for pi, p := range m.Points {
			if _, ok := finalID[slot{mi, pi}]; ok {
				continue
			}
			key := fingerprint(p.Name, p.Kind)
			if ids := oldPrint[key]; len(ids) == 1 && newPrint[key] == 1 && !used[ids[0]] {
				used[ids[0]] = true
				finalID[slot{mi, pi}] = ids[0]
				continue
			}
			finalID[slot{mi, pi}] = newKey("kp")
			change.Added++
		}
	}
	change.Kept = len(used)
	for _, p := range st.Points {
		if used[p.ID] {
			continue
		}
		change.Removed++
		if p.mastered() {
			change.RemovedMastered++
		}
	}
	if keepMastered && change.RemovedMastered > 0 {
		return OutlineChange{}, fmt.Errorf("this change would remove %d mastered knowledge point(s); mastered points stay on the map", change.RemovedMastered)
	}

	// 前置关系可以写 id、这次请求里的 ref，或唯一的名称。
	byRef := map[string]string{}
	byName := map[string][]string{}
	for mi, m := range incoming {
		for pi, p := range m.Points {
			id := finalID[slot{mi, pi}]
			if p.Ref != "" {
				byRef[p.Ref] = id
			}
			byName[strings.ToLower(p.Name)] = append(byName[strings.ToLower(p.Name)], id)
		}
	}
	known := map[string]bool{}
	for _, id := range finalID {
		known[id] = true
	}
	edges := map[string][]string{}
	names := map[string]string{}
	for mi, m := range incoming {
		for pi, p := range m.Points {
			id := finalID[slot{mi, pi}]
			names[id] = p.Name
			if p.Prereqs == nil {
				for _, pre := range oldByID[id].Prereqs {
					if known[pre] {
						edges[id] = append(edges[id], pre)
					}
				}
				continue
			}
			seen := map[string]bool{}
			for _, raw := range p.Prereqs {
				target := ""
				switch {
				case known[raw]:
					target = raw
				case byRef[raw] != "":
					target = byRef[raw]
				case len(byName[strings.ToLower(raw)]) == 1:
					target = byName[strings.ToLower(raw)][0]
				default:
					return OutlineChange{}, fmt.Errorf("knowledge point %q lists an unknown prerequisite %q", p.Name, raw)
				}
				if target == id {
					return OutlineChange{}, fmt.Errorf("knowledge point %q cannot be its own prerequisite", p.Name)
				}
				if !seen[target] {
					seen[target] = true
					edges[id] = append(edges[id], target)
				}
			}
		}
	}
	if cycle := findCycle(edges); len(cycle) > 0 {
		for i, id := range cycle {
			cycle[i] = names[id]
		}
		return OutlineChange{}, fmt.Errorf("prerequisite cycle: %s", strings.Join(cycle, " -> "))
	}

	for _, p := range st.Points {
		if used[p.ID] {
			continue
		}
		if err := execArgs(tx, p.ID,
			`DELETE FROM evidence WHERE point_id = ?`,
			`DELETE FROM interactions WHERE point_id = ?`,
			`DELETE FROM repetitions WHERE point_id = ?`,
			`DELETE FROM point_prereqs WHERE point_id = ?`,
			`DELETE FROM point_prereqs WHERE prereq_id = ?`,
			`DELETE FROM points WHERE id = ?`,
		); err != nil {
			return OutlineChange{}, err
		}
	}
	oldModules := map[string]bool{}
	modByName := map[string]string{}
	for _, m := range st.Modules {
		oldModules[m.ID] = true
		modByName[strings.ToLower(m.Name)] = m.ID
	}
	keptModules := map[string]bool{}
	for mi, m := range incoming {
		mid := m.ID
		if !oldModules[mid] || keptModules[mid] {
			mid = modByName[strings.ToLower(m.Name)]
		}
		if mid == "" || keptModules[mid] {
			mid = newKey("mod")
			if _, err := tx.Exec(`INSERT INTO modules (id, path_id, name, objective, ord) VALUES (?, ?, ?, ?, ?)`, mid, st.ID, m.Name, m.Objective, mi); err != nil {
				return OutlineChange{}, err
			}
		} else if _, err := tx.Exec(`UPDATE modules SET name = ?, objective = ?, ord = ? WHERE id = ?`, m.Name, m.Objective, mi, mid); err != nil {
			return OutlineChange{}, err
		}
		keptModules[mid] = true
		for pi, p := range m.Points {
			id := finalID[slot{mi, pi}]
			if used[id] {
				if _, err := tx.Exec(`UPDATE points SET module_id = ?, name = ?, ord = ? WHERE id = ?`, mid, p.Name, pi, id); err != nil {
					return OutlineChange{}, err
				}
				continue
			}
			if _, err := tx.Exec(`INSERT INTO points (id, path_id, module_id, name, kind, ord) VALUES (?, ?, ?, ?, ?, ?)`, id, st.ID, mid, p.Name, p.Kind, pi); err != nil {
				return OutlineChange{}, err
			}
		}
	}
	for _, m := range st.Modules {
		if !keptModules[m.ID] {
			if _, err := tx.Exec(`DELETE FROM modules WHERE id = ?`, m.ID); err != nil {
				return OutlineChange{}, err
			}
		}
	}
	if _, err := tx.Exec(`DELETE FROM point_prereqs WHERE point_id IN (SELECT id FROM points WHERE path_id = ?)`, st.ID); err != nil {
		return OutlineChange{}, err
	}
	for id, list := range edges {
		for _, pre := range list {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO point_prereqs (point_id, prereq_id) VALUES (?, ?)`, id, pre); err != nil {
				return OutlineChange{}, err
			}
		}
	}
	change.Modules = len(incoming)
	return change, nil
}

// findCycle 在前置关系里找一个环。edges 的键是知识点，值是它的前置。没有环时返回 nil，有环时返回环上的 id，首尾相同。
func findCycle(edges map[string][]string) []string {
	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var stack []string
	var walk func(id string) []string
	walk = func(id string) []string {
		state[id] = visiting
		stack = append(stack, id)
		for _, next := range edges[id] {
			switch state[next] {
			case visiting:
				for i, s := range stack {
					if s == next {
						return append(append([]string{}, stack[i:]...), next)
					}
				}
			case 0:
				if cycle := walk(next); cycle != nil {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
		return nil
	}
	for id := range edges {
		if state[id] == 0 {
			if cycle := walk(id); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}

// OutlineEdit 是对大纲的一次零碎修改。各项按 Remove、Rename、Add、Order 的顺序生效。
type OutlineEdit struct {
	Add    []OutlineAdd    `json:"add,omitempty"`
	Rename []OutlineRename `json:"rename,omitempty"`
	Remove []string        `json:"remove,omitempty"`
	Order  []string        `json:"order,omitempty"`
}

// OutlineAdd 往一章里加一个知识点。ModuleID 和 Module 二选一，Module 是章名，没有这一章就新建。
type OutlineAdd struct {
	ModuleID  string   `json:"module_id,omitempty"`
	Module    string   `json:"module,omitempty"`
	Objective string   `json:"objective,omitempty"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Prereqs   []string `json:"prereqs,omitempty"`
}

// OutlineRename 改一个知识点或一章的名字。ID 是知识点或章的 id。Objective 只对章有效。
type OutlineRename struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Objective string `json:"objective,omitempty"`
}

// ReviseOutline 对大纲做零碎修改，不动没提到的部分。keepMastered 为真时已掌握的知识点不能删。
// Order 必须是全部章 id 的一个排列。
func (s *Store) ReviseOutline(pathID string, edit OutlineEdit, keepMastered bool) (OutlineChange, error) {
	outline, err := s.Outline(pathID)
	if err != nil {
		return OutlineChange{}, err
	}
	remove := map[string]bool{}
	for _, id := range edit.Remove {
		remove[strings.TrimSpace(id)] = true
	}
	found := map[string]bool{}
	for mi := range outline {
		kept := outline[mi].Points[:0]
		for _, p := range outline[mi].Points {
			if remove[p.ID] {
				found[p.ID] = true
				continue
			}
			kept = append(kept, p)
		}
		outline[mi].Points = kept
	}
	for id := range remove {
		if !found[id] {
			return OutlineChange{}, fmt.Errorf("cannot remove %q: no such knowledge point on this path", id)
		}
	}
	for _, r := range edit.Rename {
		hit := false
		name := strings.TrimSpace(r.Name)
		for mi := range outline {
			if outline[mi].ID == r.ID {
				hit = true
				if name != "" {
					outline[mi].Name = name
				}
				if strings.TrimSpace(r.Objective) != "" {
					outline[mi].Objective = strings.TrimSpace(r.Objective)
				}
			}
			for pi := range outline[mi].Points {
				if outline[mi].Points[pi].ID == r.ID && name != "" {
					hit = true
					outline[mi].Points[pi].Name = name
				}
			}
		}
		if !hit {
			return OutlineChange{}, fmt.Errorf("cannot rename %q: no such module or knowledge point on this path", r.ID)
		}
	}
	for _, a := range edit.Add {
		if strings.TrimSpace(a.Name) == "" {
			return OutlineChange{}, fmt.Errorf("a new knowledge point needs a name")
		}
		pt := OutlinePoint{Name: a.Name, Kind: a.Kind, Prereqs: a.Prereqs}
		if pt.Prereqs == nil {
			pt.Prereqs = []string{}
		}
		at := -1
		for mi := range outline {
			if (a.ModuleID != "" && outline[mi].ID == a.ModuleID) || (a.ModuleID == "" && strings.EqualFold(outline[mi].Name, strings.TrimSpace(a.Module))) {
				at = mi
			}
		}
		if at < 0 {
			if strings.TrimSpace(a.Module) == "" {
				return OutlineChange{}, fmt.Errorf("cannot add %q: name an existing module_id or a module name", a.Name)
			}
			outline = append(outline, OutlineModule{Name: a.Module, Objective: a.Objective})
			at = len(outline) - 1
		}
		outline[at].Points = append(outline[at].Points, pt)
	}
	if len(edit.Order) > 0 {
		if len(edit.Order) != len(outline) {
			return OutlineChange{}, fmt.Errorf("order must list every module id exactly once")
		}
		byID := map[string]OutlineModule{}
		for _, m := range outline {
			byID[m.ID] = m
		}
		next := make([]OutlineModule, 0, len(outline))
		seen := map[string]bool{}
		for _, id := range edit.Order {
			m, ok := byID[id]
			if !ok || seen[id] {
				return OutlineChange{}, fmt.Errorf("order must list every module id exactly once")
			}
			seen[id] = true
			next = append(next, m)
		}
		outline = next
	}
	return s.ReplaceOutline(pathID, outline, OutlineOptions{KeepMastered: keepMastered})
}
