// 库结构的迁移。每一步只跑一次，按编号从小到大，和登记顺序无关。
// 这里只管加表、加列和把旧数据抄成新形状。掌握路径的证据回填在 mastery_migrate.go。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// migration 是一步迁移。run 和登记这一步的那一行在同一个事务里。
type migration struct {
	id   int
	name string
	run  func(tx *sql.Tx) error
}

var migrations []migration

// registerMigration 登记一步迁移。id 重复时直接 panic，这样测试一跑就能发现。
// 编号段：1 到 99 是旧补丁，100 段是掌握路径，200 段是练习，300 段是文库和阅读，400 段是课本。
func registerMigration(id int, name string, run func(tx *sql.Tx) error) {
	for _, m := range migrations {
		if m.id == id {
			panic(fmt.Sprintf("learn: duplicate migration %d (%s, %s)", id, m.name, name))
		}
	}
	migrations = append(migrations, migration{id: id, name: name, run: run})
}

// runMigrations 跑完所有还没跑过的迁移。任何一步失败就停，那一步的改动回滚。
func (s *Store) runMigrations() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (id INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	done := map[int]bool{}
	rows, err := s.db.Query(`SELECT id FROM schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			done[id] = true
		}
	}
	rows.Close()
	list := append([]migration(nil), migrations...)
	sort.Slice(list, func(i, j int) bool { return list[i].id < list[j].id })
	for _, m := range list {
		if done[m.id] {
			continue
		}
		err := s.inTx(func(tx *sql.Tx) error {
			if err := m.run(tx); err != nil {
				return err
			}
			_, err := tx.Exec(`INSERT INTO schema_migrations (id, name, applied_at) VALUES (?, ?, ?)`, m.id, m.name, time.Now().UTC().Format(time.RFC3339))
			return err
		})
		if err != nil {
			return fmt.Errorf("learn migration %d %s: %w", m.id, m.name, err)
		}
	}
	return nil
}

// inTx 在一个事务里跑 fn。fn 返回错误或 panic 时回滚。调用方要已经持有 s.mu，或者还在 Open 里。
// 库只有一条连接，fn 里每次查询都要先读完再发下一条语句。
func (s *Store) inTx(fn func(tx *sql.Tx) error) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	return fn(tx)
}

// tableColumns 返回 table 现有的列名。
func tableColumns(tx *sql.Tx, table string) (map[string]bool, error) {
	rows, err := tx.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// addColumns 给 table 补上还没有的列。cols 每一项是「列名 类型和默认值」，已经有的列跳过。
func addColumns(tx *sql.Tx, table string, cols ...string) error {
	have, err := tableColumns(tx, table)
	if err != nil {
		return err
	}
	for _, spec := range cols {
		fields := strings.Fields(spec)
		if len(fields) == 0 || have[fields[0]] {
			continue
		}
		if _, err := tx.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + spec); err != nil {
			return fmt.Errorf("%s.%s: %w", table, fields[0], err)
		}
	}
	return nil
}

// execAll 依次执行 stmts，遇到第一条失败就返回。
func execAll(tx *sql.Tx, stmts ...string) error {
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	registerMigration(1, "practice_streak", func(tx *sql.Tx) error {
		return addColumns(tx, "practice_state", "streak INTEGER NOT NULL DEFAULT 0")
	})
	registerMigration(100, "mastery_v2_schema", masterySchemaV2)
	registerMigration(200, "practice_v2_schema", practiceSchemaV2)
	registerMigration(201, "practice_v2_fsrs", practiceBackfillFSRS)
	registerMigration(300, "library_v2_schema", librarySchemaV2)
	registerMigration(400, "books_v2_schema", booksSchemaV2)
	registerMigration(401, "books_v2_chapters", booksBackfillChapters)
}

// masterySchemaV2 给路径加上目标、偏好和目标记住率，给知识点加上记忆状态和前置关系，并建证据表。
func masterySchemaV2(tx *sql.Tx) error {
	if err := addColumns(tx, "paths",
		"goal TEXT NOT NULL DEFAULT ''",
		"description TEXT NOT NULL DEFAULT ''",
		"status TEXT NOT NULL DEFAULT 'active'",
		"preferences TEXT NOT NULL DEFAULT ''",
		"desired_retention REAL NOT NULL DEFAULT 0.9",
		"revision INTEGER NOT NULL DEFAULT 0",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "points",
		"stability REAL NOT NULL DEFAULT 0",
		"difficulty REAL NOT NULL DEFAULT 0",
		"reps INTEGER NOT NULL DEFAULT 0",
		"lapses INTEGER NOT NULL DEFAULT 0",
		"last_review_at TEXT NOT NULL DEFAULT ''",
		"due_at TEXT NOT NULL DEFAULT ''",
		"mastered_at TEXT NOT NULL DEFAULT ''",
		"deferred INTEGER NOT NULL DEFAULT 0",
		"claimed INTEGER NOT NULL DEFAULT 0",
		"status TEXT NOT NULL DEFAULT 'new'",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "interactions",
		"purpose TEXT NOT NULL DEFAULT ''",
		"difficulty TEXT NOT NULL DEFAULT ''",
		"accepted_json TEXT NOT NULL DEFAULT '[]'",
		"hinted INTEGER NOT NULL DEFAULT 0",
		"result TEXT NOT NULL DEFAULT ''",
		"feedback TEXT NOT NULL DEFAULT ''",
		"graded_at TEXT NOT NULL DEFAULT ''",
		"told INTEGER NOT NULL DEFAULT 0",
	); err != nil {
		return err
	}
	return execAll(tx,
		`CREATE TABLE IF NOT EXISTS path_sources (
			id TEXT PRIMARY KEY,
			path_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			ref TEXT NOT NULL DEFAULT '',
			label TEXT NOT NULL DEFAULT '',
			ord INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS path_sources_path ON path_sources(path_id, ord)`,
		`CREATE TABLE IF NOT EXISTS point_prereqs (
			point_id TEXT NOT NULL,
			prereq_id TEXT NOT NULL,
			PRIMARY KEY (point_id, prereq_id)
		)`,
		`CREATE TABLE IF NOT EXISTS evidence (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT NOT NULL UNIQUE,
			path_id TEXT NOT NULL,
			point_id TEXT NOT NULL,
			interaction_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL,
			result TEXT NOT NULL DEFAULT '',
			independent INTEGER NOT NULL DEFAULT 1,
			answer TEXT NOT NULL DEFAULT '',
			feedback TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL DEFAULT '',
			voided INTEGER NOT NULL DEFAULT 0,
			at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS evidence_point ON evidence(point_id, seq)`,
		`CREATE INDEX IF NOT EXISTS evidence_path ON evidence(path_id, seq)`,
		`CREATE INDEX IF NOT EXISTS interactions_path ON interactions(path_id, status)`,
		`CREATE INDEX IF NOT EXISTS points_path ON points(path_id, module_id, ord)`,
	)
}

// practiceSchemaV2 给题目加上来源、标签和错题标记，给日程加上 FSRS 状态，给复习记录加上幂等键。
func practiceSchemaV2(tx *sql.Tx) error {
	if err := addColumns(tx, "questions",
		"origin_key TEXT NOT NULL DEFAULT ''",
		"difficulty TEXT NOT NULL DEFAULT ''",
		"tags_json TEXT NOT NULL DEFAULT '[]'",
		"path_id TEXT NOT NULL DEFAULT ''",
		"point_id TEXT NOT NULL DEFAULT ''",
		"library_id TEXT NOT NULL DEFAULT ''",
		"document_id TEXT NOT NULL DEFAULT ''",
		"locator INTEGER NOT NULL DEFAULT 0",
		"bookmarked INTEGER NOT NULL DEFAULT 0",
		"mistake INTEGER NOT NULL DEFAULT 0",
		"first_wrong_at TEXT NOT NULL DEFAULT ''",
		"last_answer TEXT NOT NULL DEFAULT ''",
		"updated_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "practice_state",
		"stability REAL NOT NULL DEFAULT 0",
		"difficulty REAL NOT NULL DEFAULT 0",
		"last_review_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "practice_events",
		"request_id TEXT NOT NULL DEFAULT ''",
		"answer TEXT NOT NULL DEFAULT ''",
		"correct INTEGER NOT NULL DEFAULT -1",
		"elapsed_days REAL NOT NULL DEFAULT 0",
		"stability REAL NOT NULL DEFAULT 0",
		"difficulty REAL NOT NULL DEFAULT 0",
	); err != nil {
		return err
	}
	return execAll(tx,
		`CREATE UNIQUE INDEX IF NOT EXISTS questions_origin ON questions(origin_key) WHERE origin_key != ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS practice_events_request ON practice_events(request_id) WHERE request_id != ''`,
		`CREATE INDEX IF NOT EXISTS practice_events_at ON practice_events(at)`,
		`CREATE INDEX IF NOT EXISTS practice_state_due ON practice_state(suspended, due_at)`,
		`CREATE TABLE IF NOT EXISTS practice_imports (
			token TEXT PRIMARY KEY,
			filename TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			committed INTEGER NOT NULL DEFAULT 0
		)`,
	)
}

// practiceBackfillFSRS 把旧的 SM-2 间隔和易度折成 FSRS 状态。从没复习过的题保持未见，第一次评分时再取初始值。
// 来自掌握路径的旧题本来就是答错才进来的，标成错题。
func practiceBackfillFSRS(tx *sql.Tx) error {
	type row struct {
		id       string
		interval float64
		ease     float64
		reviews  int
		due      string
	}
	rows, err := tx.Query(`SELECT question_id, interval_days, ease, review_count, due_at FROM practice_state`)
	if err != nil {
		return err
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.interval, &r.ease, &r.reviews, &r.due); err != nil {
			rows.Close()
			return err
		}
		list = append(list, r)
	}
	rows.Close()
	for _, r := range list {
		if r.reviews <= 0 {
			continue
		}
		m := MemoryFromSM2(r.ease, r.interval)
		if !m.Seen() {
			continue
		}
		last := ""
		if due, err := time.Parse(time.RFC3339Nano, r.due); err == nil {
			last = due.Add(-time.Duration(r.interval * 24 * float64(time.Hour))).UTC().Format(time.RFC3339Nano)
		}
		if _, err := tx.Exec(`UPDATE practice_state SET stability = ?, difficulty = ?, last_review_at = ? WHERE question_id = ?`,
			m.Stability, m.Difficulty, last, r.id); err != nil {
			return err
		}
	}
	return execAll(tx,
		`UPDATE questions SET mistake = 1, first_wrong_at = created_at WHERE source = 'mastery' AND mistake = 0`,
		`UPDATE questions SET updated_at = created_at WHERE updated_at = ''`,
	)
}

// librarySchemaV2 给文库加上嵌入连接和进度，给文档加上来源和提纲，给批注加上定位信息，并建阅读位置和书签。
func librarySchemaV2(tx *sql.Tx) error {
	if err := addColumns(tx, "libraries",
		"description TEXT NOT NULL DEFAULT ''",
		"embed_conn TEXT NOT NULL DEFAULT ''",
		"progress_json TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "documents",
		"source_url TEXT NOT NULL DEFAULT ''",
		"unit TEXT NOT NULL DEFAULT 'page'",
		"outline_json TEXT NOT NULL DEFAULT '[]'",
		"char_count INTEGER NOT NULL DEFAULT 0",
		"updated_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "chunks", "heading TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumns(tx, "materials", "last_opened_at TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumns(tx, "annotations",
		"kind TEXT NOT NULL DEFAULT 'note'",
		"color TEXT NOT NULL DEFAULT 'yellow'",
		"start_off INTEGER NOT NULL DEFAULT -1",
		"end_off INTEGER NOT NULL DEFAULT -1",
		"prefix TEXT NOT NULL DEFAULT ''",
		"suffix TEXT NOT NULL DEFAULT ''",
		"rects_json TEXT NOT NULL DEFAULT '[]'",
		"author TEXT NOT NULL DEFAULT 'user'",
		"source_hash TEXT NOT NULL DEFAULT ''",
		"resolution TEXT NOT NULL DEFAULT 'resolved'",
		"updated_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	return execAll(tx,
		`CREATE TABLE IF NOT EXISTS embed_cache (
			sig TEXT NOT NULL,
			text_sha TEXT NOT NULL,
			dim INTEGER NOT NULL,
			vec BLOB NOT NULL,
			PRIMARY KEY (sig, text_sha)
		)`,
		`CREATE TABLE IF NOT EXISTS reading_positions (
			material_id TEXT PRIMARY KEY,
			locator INTEGER NOT NULL DEFAULT 1,
			percent REAL NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS bookmarks (
			id TEXT PRIMARY KEY,
			material_id TEXT NOT NULL,
			locator INTEGER NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE (material_id, locator)
		)`,
		`CREATE INDEX IF NOT EXISTS annotations_material ON annotations(material_id, page_no)`,
		`CREATE INDEX IF NOT EXISTS chunks_doc ON chunks(library_id, gen, document_id, ordinal)`,
		`CREATE INDEX IF NOT EXISTS documents_library ON documents(library_id, removed)`,
		`UPDATE annotations SET kind = CASE WHEN quote != '' THEN 'highlight' ELSE 'note' END, updated_at = created_at WHERE updated_at = ''`,
	)
}

// booksSchemaV2 给课本加上意图、提案和修订号，把章节单独建表，给块加上状态、参数和载荷。
func booksSchemaV2(tx *sql.Tx) error {
	if err := addColumns(tx, "books",
		"intent TEXT NOT NULL DEFAULT ''",
		"language TEXT NOT NULL DEFAULT ''",
		"description TEXT NOT NULL DEFAULT ''",
		"proposal_json TEXT NOT NULL DEFAULT ''",
		"exploration_json TEXT NOT NULL DEFAULT ''",
		"pause_reason TEXT NOT NULL DEFAULT ''",
		"auto_compile INTEGER NOT NULL DEFAULT 1",
		"revision INTEGER NOT NULL DEFAULT 0",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "book_blocks",
		"chapter_id TEXT NOT NULL DEFAULT ''",
		"status TEXT NOT NULL DEFAULT 'ready'",
		"title TEXT NOT NULL DEFAULT ''",
		"params_json TEXT NOT NULL DEFAULT '{}'",
		"payload_json TEXT NOT NULL DEFAULT '{}'",
		"error TEXT NOT NULL DEFAULT ''",
		"edited INTEGER NOT NULL DEFAULT 0",
		"updated_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	if err := addColumns(tx, "book_progress",
		"chapter_id TEXT NOT NULL DEFAULT ''",
		"visited_json TEXT NOT NULL DEFAULT '[]'",
		"updated_at TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}
	return execAll(tx,
		`CREATE TABLE IF NOT EXISTS book_chapters (
			id TEXT PRIMARY KEY,
			book_id TEXT NOT NULL,
			ord INTEGER NOT NULL,
			title TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			objectives_json TEXT NOT NULL DEFAULT '[]',
			content_type TEXT NOT NULL DEFAULT 'concept',
			query TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			error TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS book_chapters_book ON book_chapters(book_id, ord)`,
		`CREATE INDEX IF NOT EXISTS book_blocks_chapter ON book_blocks(chapter_id, ord)`,
		`CREATE TABLE IF NOT EXISTS book_attempts (
			id TEXT PRIMARY KEY,
			book_id TEXT NOT NULL,
			block_id TEXT NOT NULL,
			question_id TEXT NOT NULL,
			answer TEXT NOT NULL DEFAULT '',
			correct INTEGER NOT NULL DEFAULT -1,
			at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS book_attempts_block ON book_attempts(book_id, block_id, question_id)`,
	)
}

// booksBackfillChapters 把旧课本抄成章节和带载荷的块。旧的章节标题块变成章节行，正文块保留原文。
// 书脊里没编出内容的那几节成为待编的章节。正在编译的旧书停成暂停，由人决定要不要接着编。
func booksBackfillChapters(tx *sql.Tx) error {
	type book struct{ id, spine, status string }
	rows, err := tx.Query(`SELECT id, spine_json, status FROM books`)
	if err != nil {
		return err
	}
	var books []book
	for rows.Next() {
		var b book
		if err := rows.Scan(&b.id, &b.spine, &b.status); err != nil {
			rows.Close()
			return err
		}
		books = append(books, b)
	}
	rows.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, b := range books {
		type block struct {
			id, kind, body string
			page           int
		}
		brows, err := tx.Query(`SELECT id, kind, body, page_no FROM book_blocks WHERE book_id = ? AND chapter_id = '' ORDER BY ord`, b.id)
		if err != nil {
			return err
		}
		var blocks []block
		for brows.Next() {
			var blk block
			if err := brows.Scan(&blk.id, &blk.kind, &blk.body, &blk.page); err != nil {
				brows.Close()
				return err
			}
			blocks = append(blocks, blk)
		}
		brows.Close()
		var spine []struct {
			Title string `json:"title"`
			Query string `json:"query"`
		}
		_ = json.Unmarshal([]byte(b.spine), &spine)
		queryOf := map[string]string{}
		for _, item := range spine {
			queryOf[strings.TrimSpace(item.Title)] = item.Query
		}
		ord := 0
		seen := map[string]bool{}
		chapterOfPage := map[int]string{}
		current := ""
		blockOrd := 0
		for _, blk := range blocks {
			if blk.kind == "section" || current == "" {
				title := strings.TrimSpace(blk.body)
				if blk.kind != "section" || title == "" {
					title = "未命名"
				}
				current = fmt.Sprintf("ch-%s-%d", b.id, ord)
				if _, err := tx.Exec(`INSERT INTO book_chapters (id, book_id, ord, title, query, status, updated_at) VALUES (?, ?, ?, ?, ?, 'ready', ?)`,
					current, b.id, ord, title, queryOf[title], now); err != nil {
					return err
				}
				seen[title] = true
				chapterOfPage[blk.page] = current
				ord++
				blockOrd = 0
				if blk.kind == "section" {
					if _, err := tx.Exec(`DELETE FROM book_blocks WHERE id = ?`, blk.id); err != nil {
						return err
					}
					continue
				}
			}
			kind := "text"
			payload := map[string]any{"body": blk.body}
			if blk.kind == "code" {
				kind = "code"
				payload = map[string]any{"language": "", "code": blk.body}
			}
			raw, _ := json.Marshal(payload)
			if _, err := tx.Exec(`UPDATE book_blocks SET chapter_id = ?, kind = ?, ord = ?, status = 'ready', payload_json = ?, updated_at = ? WHERE id = ?`,
				current, kind, blockOrd, string(raw), now, blk.id); err != nil {
				return err
			}
			blockOrd++
		}
		for _, item := range spine {
			title := strings.TrimSpace(item.Title)
			if title == "" || seen[title] {
				continue
			}
			id := fmt.Sprintf("ch-%s-%d", b.id, ord)
			if _, err := tx.Exec(`INSERT INTO book_chapters (id, book_id, ord, title, query, status, updated_at) VALUES (?, ?, ?, ?, ?, 'pending', ?)`,
				id, b.id, ord, title, item.Query, now); err != nil {
				return err
			}
			seen[title] = true
			ord++
		}
		for page, chapter := range chapterOfPage {
			if _, err := tx.Exec(`UPDATE book_progress SET chapter_id = ?, updated_at = ? WHERE book_id = ? AND page_no = ? AND chapter_id = ''`, chapter, now, b.id, page); err != nil {
				return err
			}
		}
		if b.status == "compiling" {
			if _, err := tx.Exec(`UPDATE books SET status = 'paused', pause_reason = 'migrated' WHERE id = ?`, b.id); err != nil {
				return err
			}
		}
	}
	return nil
}
