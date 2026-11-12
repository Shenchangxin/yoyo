// 掌握路径本身：建路径、档案、模式和地图视图。分数和日程在 mastery_evidence.go，下一步在 mastery_policy.go，题目在 mastery_quiz.go，大纲在 mastery_outline.go。
// 这里不调用模型。标准答案只留在库里，视图里没有。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// KindMemory 是记忆型知识点，靠答题正确率过门。
	KindMemory = "memory"
	// KindProcedure 是步骤型知识点，靠答题正确率过门。
	KindProcedure = "procedure"
	// KindConcept 是概念型知识点，靠用自己的话讲清楚过门。
	KindConcept = "concept"
	// KindDesign 是设计型知识点，靠用自己的话讲清楚过门。
	KindDesign = "design"

	// ModeOutline 是定大纲的模式。只有这个模式能改地图。
	ModeOutline = "outline"
	// ModeStudy 是学新内容的模式。
	ModeStudy = "study"
	// ModeReview 是回头检验已掌握内容的模式。
	ModeReview = "review"

	// TeachProbe 表示先用一道题摸底再讲。
	TeachProbe = "probe_first"
	// TeachFirst 表示先讲再检查。
	TeachFirst = "teach_first"

	// StatusNew 表示这个知识点还没碰过。
	StatusNew = "new"
	// StatusLearning 表示碰过但还没过门。
	StatusLearning = "learning"
	// StatusMastered 表示已经过门，或学习者自己声明会了。
	StatusMastered = "mastered"

	// quantitativeGate 是记忆和步骤型的过门分。概念和设计看讲解是否通过，不看这个数。
	quantitativeGate = 0.9

	maxModules         = 30
	maxPointsPerModule = 24
	maxNameRunes       = 120
)

var recencyWeights = []float64{0.5, 0.7, 0.85, 0.95, 1.0}

var confidenceCap = map[int]float64{1: 0.5, 2: 0.8}

// querier 是 *sql.DB 和 *sql.Tx 共有的那几样，读写函数两边都能用。
type querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// ComputeMastery 用最近五次独立作答算 0 到 1 的掌握分。只答一次封顶 0.5，两次封顶 0.8，所以至少三次才可能过 0.9 的门。
// credits 按时间从旧到新，答对是 1，部分对是 0.5，答错是 0。空列表返回 0。
func ComputeMastery(credits []float64) float64 {
	if len(credits) == 0 {
		return 0
	}
	recent := credits
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
	if limit, ok := confidenceCap[len(recent)]; ok && score > limit {
		return limit
	}
	return score
}

// qualitativeKind 在 kind 是概念或设计时为真。这两类靠讲解过门。
func qualitativeKind(kind string) bool {
	return kind == KindConcept || kind == KindDesign
}

// normalizeKind 把 kind 收成四种之一。认不出的写成概念。
func normalizeKind(kind string) string {
	switch k := strings.ToLower(strings.TrimSpace(kind)); k {
	case KindMemory, KindProcedure, KindConcept, KindDesign:
		return k
	default:
		return KindConcept
	}
}

// Counts 是一条路径上各状态的知识点数。
type Counts struct {
	Total    int `json:"total"`
	Mastered int `json:"mastered"`
	Learning int `json:"learning"`
	New      int `json:"new"`
}

// PathSource 是路径建立时选的一份材料。Kind 是 library、document 或 note，Ref 是对应的 id。
type PathSource struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Ref   string `json:"ref"`
	Label string `json:"label"`
}

// Profile 是这条路径上学习者的情况。导师据此调整讲法，不据此给分。
type Profile struct {
	Teaching       string `json:"teaching"`
	PriorKnowledge string `json:"prior_knowledge"`
	TargetLevel    string `json:"target_level"`
	TimeBudget     string `json:"time_budget"`
	Preferences    string `json:"preferences"`
}

// PathSummary 是列表里的一条路径。
type PathSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Goal        string `json:"goal,omitempty"`
	Description string `json:"description,omitempty"`
	LibraryID   string `json:"library_id,omitempty"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	Counts      Counts `json:"counts"`
	Due         int    `json:"due_reviews"`
	Pending     bool   `json:"pending_question"`
	Complete    bool   `json:"complete"`
	UpdatedAt   string `json:"updated_at"`
}

// PathView 是一条路径的完整地图。Next 里的题目没有标准答案。
type PathView struct {
	PathSummary
	Profile      Profile      `json:"profile"`
	IntakeNeeded bool         `json:"intake_needed"`
	Retention    float64      `json:"desired_retention"`
	Revision     int          `json:"revision"`
	Sources      []PathSource `json:"sources"`
	Modules      []ModuleView `json:"modules"`
	Next         *NextStep    `json:"next,omitempty"`
	Reviews      []ReviewItem `json:"reviews"`
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

// PointView 是一个知识点。Source 为 learner 表示学习者自己声明会了，不是测评证据。
// Blocked 为真表示还有前置知识点没过门。Retrievability 只在已掌握时有意义，是现在还记得的估计概率。
type PointView struct {
	ID             string   `json:"id"`
	ModuleID       string   `json:"module_id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Status         string   `json:"status"`
	Mastery        float64  `json:"mastery"`
	Source         string   `json:"mastery_source,omitempty"`
	Gate           string   `json:"gate"`
	Prereqs        []string `json:"prereqs,omitempty"`
	Blocked        bool     `json:"blocked,omitempty"`
	Deferred       bool     `json:"deferred,omitempty"`
	Attempts       int      `json:"attempts"`
	DueAt          string   `json:"due_at,omitempty"`
	Retrievability float64  `json:"retrievability,omitempty"`
}

// ReviewItem 是一条到期或即将到期的复习。
type ReviewItem struct {
	PointID        string  `json:"point_id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	DueAt          string  `json:"due_at"`
	Due            bool    `json:"due"`
	Retrievability float64 `json:"retrievability"`
}

// CreatePathRequest 是新建路径的入参。Name 为空时取 Goal 的第一句。LibraryID 和 Sources 都可空。
type CreatePathRequest struct {
	Name      string       `json:"name"`
	Goal      string       `json:"goal"`
	LibraryID string       `json:"library_id"`
	Sources   []PathSource `json:"sources"`
}

// CreatePath 新建一条大纲模式的路径。Name 和 Goal 不能都空。Name 最长 120 个符文，Goal 最长 2000 个。
func (s *Store) CreatePath(req CreatePathRequest) (PathView, error) {
	if s == nil {
		return PathView{}, fmt.Errorf("learn store unavailable")
	}
	goal := clipRunes(strings.TrimSpace(req.Goal), 2000)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = nameFromGoal(goal)
	}
	if name == "" {
		return PathView{}, fmt.Errorf("say what you want to master")
	}
	name = clipRunes(name, maxNameRunes)
	now := s.clock().Format(time.RFC3339Nano)
	id := newKey("path")
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO paths (id, library_id, name, goal, mode, teaching, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, strings.TrimSpace(req.LibraryID), name, goal, ModeOutline, TeachProbe, now, now); err != nil {
			return err
		}
		for i, src := range req.Sources {
			kind := strings.TrimSpace(src.Kind)
			ref := strings.TrimSpace(src.Ref)
			if kind == "" || ref == "" || i >= 16 {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO path_sources (id, path_id, kind, ref, label, ord) VALUES (?, ?, ?, ?, ?, ?)`,
				newKey("src"), id, kind, ref, clipRunes(strings.TrimSpace(src.Label), maxNameRunes), i); err != nil {
				return err
			}
		}
		return nil
	})
	var view PathView
	if err == nil {
		view, err = s.pathView(s.db, id)
	}
	s.mu.Unlock()
	if err != nil {
		return PathView{}, err
	}
	s.notify("path", id, "created")
	return view, nil
}

// nameFromGoal 取目标的第一句当名字，最多 32 个符文。
func nameFromGoal(goal string) string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return ""
	}
	cut := strings.IndexAny(goal, "\n。.；;！!？?")
	if cut > 0 {
		goal = goal[:cut]
	}
	goal = strings.TrimSpace(goal)
	if utf8.RuneCountInString(goal) > 32 {
		goal = string([]rune(goal)[:32])
	}
	return goal
}

// Paths 按最近更新列出全部路径的摘要。
func (s *Store) Paths() []PathSummary {
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
	now := s.clock()
	out := make([]PathSummary, 0, len(ids))
	for _, id := range ids {
		st, err := loadPath(s.db, id)
		if err != nil {
			continue
		}
		out = append(out, st.summary(now))
	}
	return out
}

// Path 读取一条路径的地图、下一步和复习队列。pathID 必须存在。
func (s *Store) Path(pathID string) (PathView, error) {
	if s == nil {
		return PathView{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pathView(s.db, pathID)
}

// pathView 用 q 读出路径并拼成视图。调用方要已经持有 s.mu。
func (s *Store) pathView(q querier, pathID string) (PathView, error) {
	st, err := loadPath(q, pathID)
	if err != nil {
		return PathView{}, err
	}
	return st.view(s.clock()), nil
}

// RenamePath 改路径的名字。name 去掉空白后不能空，最长 120 个符文。
func (s *Store) RenamePath(pathID, name string) error {
	name = clipRunes(strings.TrimSpace(name), maxNameRunes)
	if name == "" {
		return fmt.Errorf("path name required")
	}
	return s.touchPathWith(pathID, "renamed", `UPDATE paths SET name = ?, updated_at = ? WHERE id = ?`, name)
}

// DeletePath 删除路径和它的章节、知识点、证据、题目与租约。答错后进了题库的题留着。
func (s *Store) DeletePath(pathID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM paths WHERE id = ?`, pathID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("path not found")
		}
		return execArgs(tx, pathID,
			`DELETE FROM point_prereqs WHERE point_id IN (SELECT id FROM points WHERE path_id = ?)`,
			`DELETE FROM evidence WHERE path_id = ?`,
			`DELETE FROM interactions WHERE path_id = ?`,
			`DELETE FROM repetitions WHERE path_id = ?`,
			`DELETE FROM points WHERE path_id = ?`,
			`DELETE FROM modules WHERE path_id = ?`,
			`DELETE FROM path_sources WHERE path_id = ?`,
			`DELETE FROM path_leases WHERE path_id = ?`,
		)
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("path", pathID, "deleted")
	}
	return err
}

// ResetPath 清掉这条路径上的作答、证据和日程，大纲留着。模式回到学习。
func (s *Store) ResetPath(pathID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	now := s.clock().Format(time.RFC3339Nano)
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE paths SET updated_at = ?, revision = revision + 1 WHERE id = ?`, now, pathID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("path not found")
		}
		return execArgs(tx, pathID,
			`DELETE FROM evidence WHERE path_id = ?`,
			`DELETE FROM interactions WHERE path_id = ?`,
			`DELETE FROM repetitions WHERE path_id = ?`,
			`UPDATE points SET mastery = 0, qualitative = 0, mastery_source = '', attempts_json = '[]', stability = 0, difficulty = 0,
				reps = 0, lapses = 0, last_review_at = '', due_at = '', mastered_at = '', deferred = 0, claimed = 0, status = 'new' WHERE path_id = ?`,
		)
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("path", pathID, "reset")
	}
	return err
}

// execArgs 用同一个参数依次执行 stmts。
func execArgs(q querier, arg string, stmts ...string) error {
	for _, stmt := range stmts {
		if _, err := q.Exec(stmt, arg); err != nil {
			return err
		}
	}
	return nil
}

// SetMode 切换路径模式。mode 只能是 outline、study 或 review。还没有知识点时不能离开大纲模式。
func (s *Store) SetMode(pathID, mode string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case ModeOutline, ModeStudy, ModeReview:
	default:
		return fmt.Errorf("unknown mode %q: use outline, study, or review", mode)
	}
	s.mu.Lock()
	var points int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM points WHERE path_id = ?`, pathID).Scan(&points)
	if mode != ModeOutline && points == 0 {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM paths WHERE id = ?`, pathID).Scan(&n)
		s.mu.Unlock()
		if n == 0 {
			return fmt.Errorf("path not found")
		}
		return fmt.Errorf("there is no outline to study yet: stay in outline and build one first")
	}
	s.mu.Unlock()
	return s.touchPathWith(pathID, "mode", `UPDATE paths SET mode = ?, updated_at = ? WHERE id = ?`, mode)
}

// Mode 返回路径当前模式。工具在调用的那一刻用它判断，不看挂上时的模式。
func (s *Store) Mode(pathID string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var mode string
	if err := s.db.QueryRow(`SELECT mode FROM paths WHERE id = ?`, pathID).Scan(&mode); err != nil {
		return "", fmt.Errorf("path not found")
	}
	return mode, nil
}

// ProfilePatch 是档案的一次修改。指针为 nil 的字段不动。
type ProfilePatch struct {
	Teaching       *string `json:"teaching"`
	PriorKnowledge *string `json:"prior_knowledge"`
	TargetLevel    *string `json:"target_level"`
	TimeBudget     *string `json:"time_budget"`
	Preferences    *string `json:"preferences"`
}

// SetProfile 合并写入学习者档案。Teaching 只能是 probe_first 或 teach_first。其余字段各自截到 600 个符文。
func (s *Store) SetProfile(pathID string, patch ProfilePatch) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	st, err := loadPath(s.db, pathID)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	p := st.profile()
	if patch.Teaching != nil {
		t := strings.TrimSpace(*patch.Teaching)
		if t == "" {
			t = TeachProbe
		}
		if t != TeachProbe && t != TeachFirst {
			return fmt.Errorf("teaching must be probe_first or teach_first")
		}
		p.Teaching = t
	}
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = clipRunes(strings.TrimSpace(*src), 600)
		}
	}
	set(&p.PriorKnowledge, patch.PriorKnowledge)
	set(&p.TargetLevel, patch.TargetLevel)
	set(&p.TimeBudget, patch.TimeBudget)
	set(&p.Preferences, patch.Preferences)
	return s.touchPathWith(pathID, "profile",
		`UPDATE paths SET teaching = ?, prior_knowledge = ?, target_level = ?, time_budget = ?, preferences = ?, updated_at = ? WHERE id = ?`,
		p.Teaching, p.PriorKnowledge, p.TargetLevel, p.TimeBudget, p.Preferences)
}

// SetRetention 改这条路径的目标记住率，并按新的目标重排每个知识点的到期时间。不算一次复习。
// retention 先交给 ClampRetention。
func (s *Store) SetRetention(pathID string, retention float64) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	retention = ClampRetention(retention)
	now := s.clock().Format(time.RFC3339Nano)
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE paths SET desired_retention = ?, updated_at = ? WHERE id = ?`, retention, now, pathID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("path not found")
		}
		return rebuildPath(tx, pathID)
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("path", pathID, "retention")
	}
	return err
}

// touchPathWith 执行一条以 updated_at 和路径 id 收尾的更新，并发出变化通知。extra 是排在前面的参数。
func (s *Store) touchPathWith(pathID, kind, query string, extra ...any) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	args := append(append([]any(nil), extra...), s.clock().Format(time.RFC3339Nano), pathID)
	s.mu.Lock()
	res, err := s.db.Exec(query, args...)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("path not found")
	}
	s.notify("path", pathID, kind)
	return nil
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
	now := s.clock()
	var owner, until string
	err := s.db.QueryRow(`SELECT session_id, until_at FROM path_leases WHERE path_id = ?`, pathID).Scan(&owner, &until)
	if err == nil && owner != sessionID {
		if exp, parseErr := time.Parse(time.RFC3339Nano, until); parseErr == nil && exp.After(now) {
			return fmt.Errorf("path busy: another conversation is working on this path right now")
		}
	}
	exp := now.Add(2 * time.Minute).Format(time.RFC3339Nano)
	_, err = s.db.Exec(`INSERT INTO path_leases (path_id, session_id, until_at) VALUES (?, ?, ?)
		ON CONFLICT(path_id) DO UPDATE SET session_id = excluded.session_id, until_at = excluded.until_at`,
		pathID, sessionID, exp)
	return err
}

// ReleasePath 释放本会话持有的租约。sessionID 对不上时不动。
func (s *Store) ReleasePath(pathID, sessionID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`DELETE FROM path_leases WHERE path_id = ? AND session_id = ?`, pathID, sessionID)
}

// moduleRow 是库里的一章。
type moduleRow struct {
	ID, Name, Objective string
	Ord                 int
}

// pointRow 是库里的一个知识点，带上回放算出的状态。
type pointRow struct {
	ID, ModuleID, Name, Kind, Status, Source string
	Ord                                      int
	Mastery                                  float64
	Qualitative, Claimed, Deferred           bool
	Mem                                      Memory
	Reps, Lapses, Attempts                   int
	Last, Due                                time.Time
	Prereqs                                  []string
}

// mastered 在这个知识点已经过门或被声明会了时为真。
func (p pointRow) mastered() bool { return p.Status == StatusMastered }

// pathState 是一条路径读进内存后的样子。视图和下一步都从它算。
type pathState struct {
	ID, Name, Goal, Description, LibraryID, Mode, Teaching, Status string
	Prior, Level, Budget, Preferences                              string
	Retention                                                      float64
	Revision                                                       int
	UpdatedAt                                                      string
	Modules                                                        []moduleRow
	Points                                                         []pointRow
	Sources                                                        []PathSource
	Pending                                                        *storedQuestion
}

// loadPath 用 q 把路径、章节、知识点、前置关系、材料和待答题读进内存。pathID 不存在时返回 path not found。
func loadPath(q querier, pathID string) (*pathState, error) {
	st := &pathState{}
	err := q.QueryRow(`SELECT id, name, goal, description, library_id, mode, teaching, status, prior_knowledge, target_level, time_budget, preferences, desired_retention, revision, updated_at
		FROM paths WHERE id = ?`, pathID).Scan(&st.ID, &st.Name, &st.Goal, &st.Description, &st.LibraryID, &st.Mode, &st.Teaching, &st.Status,
		&st.Prior, &st.Level, &st.Budget, &st.Preferences, &st.Retention, &st.Revision, &st.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("path not found")
	}
	st.Retention = ClampRetention(st.Retention)
	rows, err := q.Query(`SELECT id, name, objective, ord FROM modules WHERE path_id = ? ORDER BY ord`, pathID)
	if err != nil {
		return nil, err
	}
	modOrd := map[string]int{}
	for rows.Next() {
		var m moduleRow
		if err := rows.Scan(&m.ID, &m.Name, &m.Objective, &m.Ord); err == nil {
			modOrd[m.ID] = len(st.Modules)
			st.Modules = append(st.Modules, m)
		}
	}
	rows.Close()
	rows, err = q.Query(`SELECT p.id, p.module_id, p.name, p.kind, p.ord, p.status, p.mastery, p.qualitative, p.mastery_source, p.claimed, p.deferred,
		p.stability, p.difficulty, p.reps, p.lapses, p.last_review_at, p.due_at, p.attempts_json
		FROM points p JOIN modules m ON m.id = p.module_id WHERE p.path_id = ? ORDER BY m.ord, p.ord`, pathID)
	if err != nil {
		return nil, err
	}
	index := map[string]int{}
	for rows.Next() {
		var p pointRow
		var qualitative, claimed, deferred int
		var last, due, attempts string
		if err := rows.Scan(&p.ID, &p.ModuleID, &p.Name, &p.Kind, &p.Ord, &p.Status, &p.Mastery, &qualitative, &p.Source, &claimed, &deferred,
			&p.Mem.Stability, &p.Mem.Difficulty, &p.Reps, &p.Lapses, &last, &due, &attempts); err != nil {
			continue
		}
		p.Qualitative, p.Claimed, p.Deferred = qualitative != 0, claimed != 0, deferred != 0
		p.Last = parseTime(last)
		p.Due = parseTime(due)
		var credits []float64
		_ = json.Unmarshal([]byte(attempts), &credits)
		p.Attempts = len(credits)
		if p.Status == "" {
			p.Status = StatusNew
		}
		index[p.ID] = len(st.Points)
		st.Points = append(st.Points, p)
	}
	rows.Close()
	rows, err = q.Query(`SELECT r.point_id, r.prereq_id FROM point_prereqs r JOIN points p ON p.id = r.point_id WHERE p.path_id = ?`, pathID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var point, prereq string
		if err := rows.Scan(&point, &prereq); err != nil {
			continue
		}
		if i, ok := index[point]; ok {
			if _, known := index[prereq]; known {
				st.Points[i].Prereqs = append(st.Points[i].Prereqs, prereq)
			}
		}
	}
	rows.Close()
	rows, err = q.Query(`SELECT id, kind, ref, label FROM path_sources WHERE path_id = ? ORDER BY ord`, pathID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var src PathSource
		if err := rows.Scan(&src.ID, &src.Kind, &src.Ref, &src.Label); err == nil {
			st.Sources = append(st.Sources, src)
		}
	}
	rows.Close()
	pending, err := pendingQuestion(q, pathID)
	if err != nil {
		return nil, err
	}
	st.Pending = pending
	return st, nil
}

// parseTime 解析 RFC3339 时间。空串或格式不对时返回零值。
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// formatTime 把时刻写成 RFC3339。零值写成空串。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// profile 返回这条路径上的学习者档案。
func (st *pathState) profile() Profile {
	teaching := st.Teaching
	if teaching != TeachFirst {
		teaching = TeachProbe
	}
	return Profile{Teaching: teaching, PriorKnowledge: st.Prior, TargetLevel: st.Level, TimeBudget: st.Budget, Preferences: st.Preferences}
}

// masteredSet 返回已过门知识点的 id 集合。
func (st *pathState) masteredSet() map[string]bool {
	out := map[string]bool{}
	for _, p := range st.Points {
		if p.mastered() {
			out[p.ID] = true
		}
	}
	return out
}

// blocked 在 p 还有前置知识点没过门时为真。done 是已过门的集合。
func blocked(p pointRow, done map[string]bool) bool {
	for _, pre := range p.Prereqs {
		if !done[pre] {
			return true
		}
	}
	return false
}

// counts 统计各状态的知识点数和到了复习时间的数量。now 用来判断到期。
func (st *pathState) counts(now time.Time) (Counts, int) {
	var c Counts
	due := 0
	for _, p := range st.Points {
		c.Total++
		switch p.Status {
		case StatusMastered:
			c.Mastered++
			if !p.Deferred && !p.Due.IsZero() && !p.Due.After(now) {
				due++
			}
		case StatusLearning:
			c.Learning++
		default:
			c.New++
		}
	}
	return c, due
}

// summary 把路径收成列表里的一条。now 用来判断到期。
func (st *pathState) summary(now time.Time) PathSummary {
	c, due := st.counts(now)
	status := st.Status
	if status == "" {
		status = "active"
	}
	return PathSummary{
		ID: st.ID, Name: st.Name, Goal: st.Goal, Description: st.Description, LibraryID: st.LibraryID,
		Mode: st.Mode, Status: status, Counts: c, Due: due, Pending: st.Pending != nil,
		Complete:  c.Total > 0 && c.Mastered == c.Total && due == 0 && st.Pending == nil,
		UpdatedAt: st.UpdatedAt,
	}
}

// view 把路径拼成完整地图，带下一步和复习队列。now 用来判断到期和估计还记得多少。
func (st *pathState) view(now time.Time) PathView {
	view := PathView{
		PathSummary: st.summary(now),
		Profile:     st.profile(),
		Retention:   st.Retention,
		Revision:    st.Revision,
		Sources:     st.Sources,
		Reviews:     st.reviews(now, 12),
	}
	if view.Sources == nil {
		view.Sources = []PathSource{}
	}
	// 四项都空说明导师还没问过学习者的情况。
	view.IntakeNeeded = st.Prior == "" && st.Level == "" && st.Budget == "" && st.Preferences == ""
	done := st.masteredSet()
	byModule := map[string][]PointView{}
	for _, p := range st.Points {
		pv := PointView{
			ID: p.ID, ModuleID: p.ModuleID, Name: p.Name, Kind: p.Kind, Status: p.Status, Mastery: p.Mastery,
			Source: p.Source, Gate: "quantitative", Prereqs: p.Prereqs, Deferred: p.Deferred, Attempts: p.Attempts,
		}
		if qualitativeKind(p.Kind) {
			pv.Gate = "qualitative"
		}
		if !p.mastered() {
			pv.Blocked = blocked(p, done)
		} else if p.Mem.Seen() {
			pv.DueAt = formatTime(p.Due)
			pv.Retrievability = Retrievability(now.Sub(p.Last).Hours()/24, p.Mem.Stability)
		}
		byModule[p.ModuleID] = append(byModule[p.ModuleID], pv)
	}
	view.Modules = make([]ModuleView, 0, len(st.Modules))
	for _, m := range st.Modules {
		mv := ModuleView{ID: m.ID, Name: m.Name, Objective: m.Objective, Order: m.Ord, Points: byModule[m.ID]}
		if mv.Points == nil {
			mv.Points = []PointView{}
		}
		for _, p := range mv.Points {
			mv.Total++
			if p.Status == StatusMastered {
				mv.Mastered++
			}
		}
		view.Modules = append(view.Modules, mv)
	}
	step := st.next(now)
	view.Next = &step
	return view
}
