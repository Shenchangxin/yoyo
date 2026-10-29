// 掌握路径的门、下一题和测验。分数在这里算完，不调用模型。expected_answer 只留在库里。

package learn

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// 定量门。记忆和步骤要到 0.9。概念和设计看定性通过，不看这张表。
const quantitativeGate = 0.9

var recencyWeights = []float64{0.5, 0.7, 0.85, 0.95, 1.0}

var confidenceCap = map[int]float64{1: 0.5, 2: 0.8}

var intervalDays = map[string][]int{
	"memory":    {0, 1, 3, 7, 14, 30, 60},
	"concept":   {3, 7, 14, 30},
	"procedure": {3, 7, 14},
	"design":    {14, 28},
}

// ComputeMastery 用最近几次对错算 0 到 1 的掌握分。一次答对封顶 0.5，两次封顶 0.8。
// correctness 按时间从旧到新。空列表返回 0。
func ComputeMastery(correctness []float64) float64 {
	if len(correctness) == 0 {
		return 0
	}
	recent := correctness
	if len(recent) > len(recencyWeights) {
		recent = recent[len(recent)-len(recencyWeights):]
	}
	weights := recencyWeights[len(recencyWeights)-len(recent):]
	var num, den float64
	for i, c := range recent {
		if c < 0 {
			c = 0
		}
		if c > 1 {
			c = 1
		}
		num += weights[i] * c
		den += weights[i]
	}
	if den == 0 {
		return 0
	}
	score := num / den
	if cap, ok := confidenceCap[len(recent)]; ok && score > cap {
		return cap
	}
	return score
}

// IntervalDays 返回这个知识点类型的复习间隔，单位是天。未知类型按记忆。
// kind 去掉空白并转成小写。
func IntervalDays(kind string) []int {
	kind = strings.ToLower(strings.TrimSpace(kind))
	seq, ok := intervalDays[kind]
	if !ok {
		return intervalDays["memory"]
	}
	out := make([]int, len(seq))
	copy(out, seq)
	return out
}

func qualitativeKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "concept", "design":
		return true
	default:
		return false
	}
}

// PathView 是一条路径的地图。Next 里没有标准答案。
type PathView struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Library  string         `json:"library_id,omitempty"`
	Mode     string         `json:"mode"`
	Teaching string         `json:"teaching"`
	Counts   map[string]int `json:"counts"`
	Due      int            `json:"due_reviews"`
	Complete bool           `json:"complete"`
	Modules  []ModuleView   `json:"modules"`
	Next     *NextStep      `json:"next,omitempty"`
}

// ModuleView 是路径里的一章。
type ModuleView struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Objective string      `json:"objective,omitempty"`
	Order     int         `json:"order"`
	Mastered  int         `json:"mastered"`
	Total     int         `json:"total"`
	Points    []PointView `json:"points"`
}

// PointView 是一个知识点。MasterySource 为 learner 时表示学习者自己声明，不是测评证据。
type PointView struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	Status  string  `json:"status"`
	Mastery float64 `json:"mastery"`
	Source  string  `json:"mastery_source,omitempty"`
	Gate    string  `json:"gate"`
}

// NextStep 是引擎决定的下一步。
type NextStep struct {
	Action   string          `json:"action"`
	PointID  string          `json:"point_id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Kind     string          `json:"kind,omitempty"`
	Reason   string          `json:"reason,omitempty"`
	Question *PublicQuestion `json:"question,omitempty"`
}

// PublicQuestion 是给模型和界面的题目。没有标准答案，也没有解析。
type PublicQuestion struct {
	ID      string   `json:"id"`
	PointID string   `json:"point_id"`
	Type    string   `json:"question_type"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options,omitempty"`
}

// OutlineModule 是重建大纲时的一章。Points 按顺序写入。
type OutlineModule struct {
	Name      string         `json:"name"`
	Objective string         `json:"objective,omitempty"`
	Points    []OutlinePoint `json:"points"`
}

// OutlinePoint 是大纲里的一个知识点。Kind 只能是 memory、procedure、concept、design。
type OutlinePoint struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// CreatePath 新建一条大纲模式的路径。name 去掉空白后不能空。
// libraryID 可空。空则不绑定文库。
func (s *Store) CreatePath(name, libraryID string) (PathView, error) {
	if s == nil {
		return PathView{}, fmt.Errorf("learn store unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return PathView{}, fmt.Errorf("path name required")
	}
	if utf8.RuneCountInString(name) > 80 {
		name = string([]rune(name)[:80])
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := newKey("path")
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`INSERT INTO paths (id, library_id, name, mode, teaching, created_at, updated_at) VALUES (?, ?, ?, 'outline', 'probe_first', ?, ?)`,
		id, strings.TrimSpace(libraryID), name, now, now)
	if err != nil {
		return PathView{}, err
	}
	return s.pathView(id)
}

// Paths 列出全部路径的摘要。摘要不含待答题目的答案。
func (s *Store) Paths() []PathView {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id FROM paths ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	var out []PathView
	for _, id := range ids {
		view, err := s.pathView(id)
		if err == nil {
			view.Next = nil
			out = append(out, view)
		}
	}
	return out
}

// Path 读取一条路径，并带上下一步。pathID 必须存在。
func (s *Store) Path(pathID string) (PathView, error) {
	if s == nil {
		return PathView{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	view, err := s.pathView(pathID)
	if err != nil {
		return PathView{}, err
	}
	step := s.nextStep(pathID)
	view.Next = &step
	return view, nil
}

// ReplaceOutline 整份替换模块和知识点。旧的待答题和复习日程一并清掉。
// pathID 必须存在。modules 为空时路径变成没有知识点。
func (s *Store) ReplaceOutline(pathID string, modules []OutlineModule) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.touchPath(pathID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM interactions WHERE path_id = ?`, pathID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM repetitions WHERE path_id = ?`, pathID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM points WHERE path_id = ?`, pathID); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM modules WHERE path_id = ?`, pathID); err != nil {
		return err
	}
	for i, mod := range modules {
		mid := newKey("mod")
		name := strings.TrimSpace(mod.Name)
		if name == "" {
			name = fmt.Sprintf("模块 %d", i+1)
		}
		if _, err := s.db.Exec(`INSERT INTO modules (id, path_id, name, objective, ord) VALUES (?, ?, ?, ?, ?)`,
			mid, pathID, name, strings.TrimSpace(mod.Objective), i); err != nil {
			return err
		}
		for j, pt := range mod.Points {
			kind := strings.ToLower(strings.TrimSpace(pt.Kind))
			switch kind {
			case "memory", "procedure", "concept", "design":
			default:
				kind = "concept"
			}
			pname := strings.TrimSpace(pt.Name)
			if pname == "" {
				continue
			}
			if _, err := s.db.Exec(`INSERT INTO points (id, path_id, module_id, name, kind, ord) VALUES (?, ?, ?, ?, ?, ?)`,
				newKey("kp"), pathID, mid, pname, kind, j); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetMode 切换路径模式。mode 只能是 outline、study 或 review。
// pathID 必须存在。
func (s *Store) SetMode(pathID, mode string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "outline", "study", "review":
	default:
		return fmt.Errorf("unknown mode")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE paths SET mode = ?, updated_at = ? WHERE id = ?`, mode, time.Now().UTC().Format(time.RFC3339Nano), pathID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("path not found")
	}
	return nil
}

// Mode 返回路径当前模式。调用测验工具时用它判断，不看工具挂上时的模式。
// pathID 必须存在。
func (s *Store) Mode(pathID string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var mode string
	err := s.db.QueryRow(`SELECT mode FROM paths WHERE id = ?`, pathID).Scan(&mode)
	if err != nil {
		return "", fmt.Errorf("path not found")
	}
	return mode, nil
}

// SetProfile 写下学习偏好。teaching 空着时保留 probe_first。
// pathID 必须存在。prior、level、budget 原样去掉首尾空白后写入。
func (s *Store) SetProfile(pathID, teaching, prior, level, budget string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	teaching = strings.TrimSpace(teaching)
	if teaching == "" {
		teaching = "probe_first"
	}
	if teaching != "probe_first" && teaching != "teach_first" {
		return fmt.Errorf("unknown teaching")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE paths SET teaching = ?, prior_knowledge = ?, target_level = ?, time_budget = ?, updated_at = ? WHERE id = ?`,
		teaching, strings.TrimSpace(prior), strings.TrimSpace(level), strings.TrimSpace(budget), time.Now().UTC().Format(time.RFC3339Nano), pathID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("path not found")
	}
	return nil
}

// ClaimMastery 记下学习者自己的声明。它把 mastery_source 写成 learner，不写入测评通过。
// pathID 和 pointID 必须属于同一条路径。note 目前不入库，避免冒充证据。
func (s *Store) ClaimMastery(pathID, pointID, note string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	_ = note
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE points SET mastery_source = 'learner' WHERE id = ? AND path_id = ?`, pointID, pathID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("point not found")
	}
	return nil
}

// PoseQuiz 留下一道待答题目。已有未答题目时拒绝再出。返给调用方的结构不含标准答案。
// pathID、pointID 必须匹配。sessionID 记在交互上。qtype 空则是 short。prompt 不能空。expected 空着会在批改时判错。
func (s *Store) PoseQuiz(pathID, pointID, sessionID, qtype, prompt, expected string, options []string) (PublicQuestion, error) {
	if s == nil {
		return PublicQuestion{}, fmt.Errorf("learn store unavailable")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return PublicQuestion{}, fmt.Errorf("prompt required")
	}
	qtype = strings.ToLower(strings.TrimSpace(qtype))
	if qtype == "" {
		qtype = "short"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM points WHERE id = ? AND path_id = ?`, pointID, pathID).Scan(&n)
	if n == 0 {
		return PublicQuestion{}, fmt.Errorf("point not found")
	}
	var pending int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM interactions WHERE path_id = ? AND status = 'pending'`, pathID).Scan(&pending)
	if pending > 0 {
		return PublicQuestion{}, fmt.Errorf("a question is already waiting")
	}
	id := newKey("q")
	raw, _ := json.Marshal(options)
	_, err := s.db.Exec(`INSERT INTO interactions
		(id, path_id, point_id, session_id, status, question_type, prompt, expected_answer, options_json, created_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?)`,
		id, pathID, pointID, sessionID, qtype, prompt, strings.TrimSpace(expected), string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PublicQuestion{}, err
	}
	return PublicQuestion{ID: id, PointID: pointID, Type: qtype, Prompt: prompt, Options: options}, nil
}

// GradeQuiz 批改这条路径上待答的那一题。expected 为空时判错。答错会写进同一张练习题表。
// pathID 定位待答题。answer 是学习者提交的文本。
func (s *Store) GradeQuiz(pathID, answer string) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var id, pointID, qtype, expected, prompt string
	err := s.db.QueryRow(`SELECT id, point_id, question_type, expected_answer, prompt FROM interactions WHERE path_id = ? AND status = 'pending' ORDER BY created_at DESC LIMIT 1`, pathID).
		Scan(&id, &pointID, &qtype, &expected, &prompt)
	if err != nil {
		return nil, fmt.Errorf("no question is waiting")
	}
	ok := gradeAnswer(answer, expected, qtype)
	score := 0.0
	if ok {
		score = 1
	}
	_, err = s.db.Exec(`UPDATE interactions SET status = 'graded', user_answer = ?, score = ? WHERE id = ?`, strings.TrimSpace(answer), score, id)
	if err != nil {
		return nil, err
	}
	if err := s.recordAttempt(pathID, pointID, score); err != nil {
		return nil, err
	}
	if !ok {
		_ = s.missToQuestion(prompt, expected, id)
	}
	var mastery float64
	var source, kind string
	_ = s.db.QueryRow(`SELECT mastery, mastery_source, kind FROM points WHERE id = ?`, pointID).Scan(&mastery, &source, &kind)
	return map[string]any{
		"correct": ok, "mastery": mastery, "mastery_source": source, "point_id": pointID, "kind": kind,
	}, nil
}

// RevealGraded 返回这条路径最近一道已批改题的标准答案。给操作者看，模型工具不调用它。
// pathID 没有已批改题时返回空串。
func (s *Store) RevealGraded(pathID string) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var expected string
	_ = s.db.QueryRow(`SELECT expected_answer FROM interactions WHERE path_id = ? AND status = 'graded' ORDER BY created_at DESC LIMIT 1`, pathID).Scan(&expected)
	return expected
}

// AssessPoint 记录概念或设计的定性结果。通过写成 qualitative=1 且 mastery_source=system。失败清掉定性通过，不改学习者声明以外的分数来源冒充。
// pathID 和 pointID 必须匹配。pass 为真表示通过。
func (s *Store) AssessPoint(pathID, pointID string, pass bool) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var kind, source string
	err := s.db.QueryRow(`SELECT kind, mastery_source FROM points WHERE id = ? AND path_id = ?`, pointID, pathID).Scan(&kind, &source)
	if err != nil {
		return fmt.Errorf("point not found")
	}
	if !qualitativeKind(kind) {
		return fmt.Errorf("this point uses a score gate")
	}
	q := 0
	nextSource := source
	if pass {
		q = 1
		if source != "learner" {
			nextSource = "system"
		}
	} else if source == "system" {
		nextSource = ""
	}
	_, err = s.db.Exec(`UPDATE points SET qualitative = ?, mastery_source = ? WHERE id = ?`, q, nextSource, pointID)
	if err != nil {
		return err
	}
	if pass {
		return s.schedulePoint(pathID, pointID, kind, true)
	}
	return s.schedulePoint(pathID, pointID, kind, false)
}

// HoldPath 占住这条路径的写租约。另一条会话还没过期时返回 path busy。
// pathID 是路径。sessionID 为空时拒绝。
func (s *Store) HoldPath(pathID, sessionID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hold(pathID, sessionID)
}

// ReleasePath 释放本会话持有的租约。sessionID 对不上时不动。
// pathID 是路径。
func (s *Store) ReleasePath(pathID, sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`DELETE FROM path_leases WHERE path_id = ? AND session_id = ?`, pathID, sessionID)
}

func (s *Store) hold(pathID, sessionID string) error {
	now := time.Now().UTC()
	var owner, until string
	err := s.db.QueryRow(`SELECT session_id, until_at FROM path_leases WHERE path_id = ?`, pathID).Scan(&owner, &until)
	if err == nil && owner != sessionID {
		exp, parseErr := time.Parse(time.RFC3339Nano, until)
		if parseErr == nil && exp.After(now) {
			return fmt.Errorf("path busy")
		}
	}
	exp := now.Add(2 * time.Minute).Format(time.RFC3339Nano)
	_, err = s.db.Exec(`INSERT INTO path_leases (path_id, session_id, until_at) VALUES (?, ?, ?)
		ON CONFLICT(path_id) DO UPDATE SET session_id = excluded.session_id, until_at = excluded.until_at`,
		pathID, sessionID, exp)
	return err
}

func (s *Store) recordAttempt(pathID, pointID string, score float64) error {
	var raw, kind, source string
	var qualitative int
	err := s.db.QueryRow(`SELECT attempts_json, kind, mastery_source, qualitative FROM points WHERE id = ? AND path_id = ?`, pointID, pathID).
		Scan(&raw, &kind, &source, &qualitative)
	if err != nil {
		return err
	}
	var attempts []float64
	_ = json.Unmarshal([]byte(raw), &attempts)
	attempts = append(attempts, score)
	blob, _ := json.Marshal(attempts)
	mastery := ComputeMastery(attempts)
	next := source
	if !qualitativeKind(kind) && mastery >= quantitativeGate && source != "learner" {
		next = "system"
	}
	_, err = s.db.Exec(`UPDATE points SET attempts_json = ?, mastery = ?, mastery_source = ? WHERE id = ?`, string(blob), mastery, next, pointID)
	if err != nil {
		return err
	}
	mastered := (!qualitativeKind(kind) && mastery >= quantitativeGate) || qualitative != 0 || source == "learner"
	return s.schedulePoint(pathID, pointID, kind, mastered && score >= 1)
}

func (s *Store) schedulePoint(pathID, pointID, kind string, correct bool) error {
	seq := IntervalDays(kind)
	var idx int
	err := s.db.QueryRow(`SELECT interval_index FROM repetitions WHERE point_id = ?`, pointID).Scan(&idx)
	exists := err == nil
	if !correct {
		idx = 0
	} else if exists {
		if idx+1 < len(seq) {
			idx++
		}
	}
	if idx >= len(seq) {
		idx = len(seq) - 1
	}
	if idx < 0 {
		idx = 0
	}
	due := time.Now().UTC().Add(time.Duration(seq[idx]) * 24 * time.Hour).Format(time.RFC3339Nano)
	if exists {
		_, err = s.db.Exec(`UPDATE repetitions SET interval_index = ?, next_review_at = ? WHERE point_id = ?`, idx, due, pointID)
		return err
	}
	_, err = s.db.Exec(`INSERT INTO repetitions (point_id, path_id, interval_index, next_review_at, consecutive_correct) VALUES (?, ?, ?, ?, ?)`,
		pointID, pathID, idx, due, boolInt(correct))
	return err
}

func (s *Store) missToQuestion(prompt, expected, interactionID string) error {
	if strings.TrimSpace(expected) == "" {
		return nil
	}
	id := newKey("bank")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`INSERT INTO questions (id, stem, qtype, answer, source, source_id, created_at) VALUES (?, ?, 'short', ?, 'mastery', ?, ?)`,
		id, prompt, expected, interactionID, now)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO practice_state (question_id, due_at, interval_days, ease, lapses, review_count) VALUES (?, ?, 1, 2.5, 1, 0)`,
		id, now)
	return err
}

func (s *Store) pathView(pathID string) (PathView, error) {
	var view PathView
	err := s.db.QueryRow(`SELECT id, library_id, name, mode, teaching FROM paths WHERE id = ?`, pathID).
		Scan(&view.ID, &view.Library, &view.Name, &view.Mode, &view.Teaching)
	if err != nil {
		return PathView{}, fmt.Errorf("path not found")
	}
	view.Counts = map[string]int{"mastered": 0, "learning": 0, "new": 0, "total": 0}
	rows, err := s.db.Query(`SELECT id, name, objective, ord FROM modules WHERE path_id = ? ORDER BY ord`, pathID)
	if err != nil {
		return PathView{}, err
	}
	var mods []ModuleView
	for rows.Next() {
		var mod ModuleView
		if err := rows.Scan(&mod.ID, &mod.Name, &mod.Objective, &mod.Order); err != nil {
			continue
		}
		mods = append(mods, mod)
	}
	rows.Close()
	pointRows, err := s.db.Query(`SELECT module_id, id, name, kind, mastery, qualitative, mastery_source, attempts_json FROM points WHERE path_id = ? ORDER BY ord`, pathID)
	if err != nil {
		return PathView{}, err
	}
	byModule := map[string][]PointView{}
	for pointRows.Next() {
		var moduleID string
		var p PointView
		var qualitative int
		var raw string
		if err := pointRows.Scan(&moduleID, &p.ID, &p.Name, &p.Kind, &p.Mastery, &qualitative, &p.Source, &raw); err != nil {
			continue
		}
		p.Gate = "quantitative"
		mastered := p.Mastery >= quantitativeGate || p.Source == "learner"
		if qualitativeKind(p.Kind) {
			p.Gate = "qualitative"
			mastered = qualitative != 0 || p.Source == "learner"
			if qualitative != 0 {
				p.Mastery = 1
			}
		}
		seen := raw != "" && raw != "[]"
		switch {
		case mastered:
			p.Status = "mastered"
			if p.Source == "" {
				p.Source = "system"
			}
		case seen || qualitative != 0:
			p.Status = "learning"
		default:
			p.Status = "new"
		}
		view.Counts[p.Status]++
		view.Counts["total"]++
		byModule[moduleID] = append(byModule[moduleID], p)
	}
	pointRows.Close()
	for i := range mods {
		mods[i].Points = byModule[mods[i].ID]
		for _, p := range mods[i].Points {
			mods[i].Total++
			if p.Status == "mastered" {
				mods[i].Mastered++
			}
		}
	}
	view.Modules = mods
	view.Due = s.dueCount(pathID)
	view.Complete = view.Counts["total"] > 0 && view.Counts["mastered"] == view.Counts["total"] && view.Due == 0
	return view, nil
}

func (s *Store) dueCount(pathID string) int {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM repetitions r JOIN points p ON p.id = r.point_id
		WHERE r.path_id = ? AND r.next_review_at <= ? AND (p.mastery >= ? OR p.qualitative = 1 OR p.mastery_source = 'learner')`,
		pathID, now, quantitativeGate).Scan(&n)
	return n
}

func (s *Store) nextStep(pathID string) NextStep {
	var qid, pointID, qtype, prompt, raw string
	err := s.db.QueryRow(`SELECT id, point_id, question_type, prompt, options_json FROM interactions WHERE path_id = ? AND status = 'pending' ORDER BY created_at LIMIT 1`, pathID).
		Scan(&qid, &pointID, &qtype, &prompt, &raw)
	if err == nil {
		var options []string
		_ = json.Unmarshal([]byte(raw), &options)
		return NextStep{
			Action: "answer_pending", PointID: pointID, Reason: "先批改已经出的题",
			Question: &PublicQuestion{ID: qid, PointID: pointID, Type: qtype, Prompt: prompt, Options: options},
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var dueID, dueName, dueKind string
	err = s.db.QueryRow(`SELECT p.id, p.name, p.kind FROM repetitions r JOIN points p ON p.id = r.point_id
		WHERE r.path_id = ? AND r.next_review_at <= ? AND (p.mastery >= ? OR p.qualitative = 1 OR p.mastery_source = 'learner')
		ORDER BY r.next_review_at LIMIT 1`, pathID, now, quantitativeGate).Scan(&dueID, &dueName, &dueKind)
	if err == nil {
		return NextStep{Action: "review", PointID: dueID, Name: dueName, Kind: dueKind, Reason: "到了复习时间"}
	}
	rows, err := s.db.Query(`SELECT p.id, p.name, p.kind, p.mastery, p.qualitative, p.mastery_source, p.attempts_json, paths.teaching
		FROM points p JOIN modules m ON m.id = p.module_id JOIN paths ON paths.id = p.path_id
		WHERE p.path_id = ? ORDER BY m.ord, p.ord`, pathID)
	if err != nil {
		return NextStep{Action: "complete", Reason: "路径是空的"}
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, kind, source, attempts, teaching string
		var mastery float64
		var qualitative int
		if err := rows.Scan(&id, &name, &kind, &mastery, &qualitative, &source, &attempts, &teaching); err != nil {
			continue
		}
		mastered := source == "learner" || qualitative != 0
		if !qualitativeKind(kind) {
			mastered = source == "learner" || mastery >= quantitativeGate
		}
		if mastered {
			continue
		}
		fresh := attempts == "" || attempts == "[]"
		action := "practice"
		if fresh {
			action = "probe"
			if teaching == "teach_first" {
				action = "teach"
			}
		} else if qualitativeKind(kind) {
			action = "assess"
		}
		return NextStep{Action: action, PointID: id, Name: name, Kind: kind, Reason: "按大纲顺序做还没掌握的点"}
	}
	return NextStep{Action: "complete", Reason: "全部掌握，也没有到期复习"}
}

func (s *Store) touchPath(pathID string) error {
	res, err := s.db.Exec(`UPDATE paths SET updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), pathID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("path not found")
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func gradeAnswer(user, expected, qtype string) bool {
	user = strings.ToLower(strings.TrimSpace(user))
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return false
	}
	switch qtype {
	case "choice":
		return strings.ReplaceAll(user, " ", "") == strings.ReplaceAll(expected, " ", "")
	case "open":
		parts := strings.FieldsFunc(expected, func(r rune) bool {
			return r == ',' || r == ';' || r == '，' || r == '；' || r == '。' || r == '\n'
		})
		if len(parts) == 0 {
			return false
		}
		hit := 0
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" && strings.Contains(user, p) {
				hit++
			}
		}
		return float64(hit)/float64(len(parts)) >= 0.6
	default:
		if user == expected {
			return true
		}
		if utf8.RuneCountInString(expected) <= 30 {
			return similarity(user, expected) >= 0.85
		}
		return false
	}
}

func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	dist := levenshtein(ar, br)
	den := len(ar)
	if len(br) > den {
		den = len(br)
	}
	return 1 - float64(dist)/float64(den)
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = prev[j] + 1
			if cur[j-1]+1 < cur[j] {
				cur[j] = cur[j-1] + 1
			}
			if prev[j-1]+cost < cur[j] {
				cur[j] = prev[j-1] + cost
			}
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
