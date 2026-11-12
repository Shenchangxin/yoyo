// Package learn 是 Yoyo 的学习模块。笔记、文库、阅读、掌握路径、练习、手记和课本都在一份 SQLite 里。
// 桌面应用用 Open 打开它，并把它当作 pages.Catalog。Harbor 不调用 Open。
// 分数、日程和下一步都在这个包里算。模型只经 Complete 和 Embed 两个注入口进来，拿不到库。
package learn

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// EmbedFunc 把一批文本变成向量。model 是库上配置的嵌入模型名。返回的条数必须和 texts 一样。
type EmbedFunc func(model string, texts []string) ([][]float32, error)

// Change 说明哪一块学习数据变了。Scope 是 path、practice、library、material、book、journal 或 notes。
// ID 是那一块里的对象，可空。Kind 是变化的种类，例如 graded、indexed、compiled。
type Change struct {
	Scope string `json:"scope"`
	ID    string `json:"id,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

// Store 拥有 learn.sqlite。原文文件在 root/raw，不进库。
type Store struct {
	mu   sync.Mutex
	db   *sql.DB
	root string
	// index 记着每个库的后台索引状态，vectors 是检索用的向量缓存。两者都由 s.mu 保护。
	index   map[string]*indexJob
	vectors map[string]*vectorSet
	// books 记着每本课本正在跑的写作任务，由 s.mu 保护。
	books map[string]*bookJob
	// Embed 为空或库没配模型时，重建只写 FTS5。
	Embed EmbedFunc
	// Complete 是不带工具的一次模型补全。判分、起草大纲、出题和写课本用它。为空时这些功能返回 ErrNoModel。
	Complete CompleteFunc
	// OnChange 在一次写入提交之后调用，不在锁里。桌面用它通知界面重新读取。
	OnChange func(Change)
	// Now 返回当前时刻。测试换掉它来走日程。为空时用 time.Now。
	Now func() time.Time

	// closing 在 Close 时关掉，后台的索引和写书靠它收工。bg 等它们都退出。
	closing   chan struct{}
	closeOnce sync.Once
	bg        sync.WaitGroup
}

// spawn 在后台跑 fn，并把它算进 Close 要等的活里。fn 应当在 stop 关掉后尽快返回。
// 库已经在关时不启动，返回 false。
func (s *Store) spawn(fn func(stop <-chan struct{})) bool {
	if s == nil || s.closing == nil {
		return false
	}
	select {
	case <-s.closing:
		return false
	default:
	}
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		fn(s.closing)
	}()
	return true
}

// stopping 在库正在关闭时为真。后台的活在每一步之间看它。
func (s *Store) stopping() bool {
	if s == nil || s.closing == nil {
		return true
	}
	select {
	case <-s.closing:
		return true
	default:
		return false
	}
}

// clock 返回当前的 UTC 时刻。
func (s *Store) clock() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// notify 把一次变化交给 OnChange。调用方必须已经放开 s.mu。
func (s *Store) notify(scope, id, kind string) {
	if s == nil || s.OnChange == nil {
		return
	}
	s.OnChange(Change{Scope: scope, ID: id, Kind: kind})
}

const schema = `
CREATE TABLE IF NOT EXISTS schema_meta (version INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS flags (k TEXT PRIMARY KEY, v TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  space_id TEXT NOT NULL,
  parent_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  revision INTEGER NOT NULL,
  source_session TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS note_reviews (
  id TEXT PRIMARY KEY,
  thread_id TEXT NOT NULL DEFAULT '',
  tool_call_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  space_id TEXT NOT NULL DEFAULT '',
  page_id TEXT NOT NULL DEFAULT '',
  parent_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  expected_revision INTEGER NOT NULL DEFAULT 0,
  hash TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  saved_page_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS libraries (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  linked_path TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  embed_model TEXT NOT NULL DEFAULT '',
  embed_dim INTEGER NOT NULL DEFAULT 0,
  embed_base TEXT NOT NULL DEFAULT '',
  published_gen INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS documents (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  rel_path TEXT NOT NULL,
  title TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  bytes INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL,
  external_path TEXT NOT NULL DEFAULT '',
  page_count INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  removed INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  UNIQUE(library_id, rel_path)
);

CREATE TABLE IF NOT EXISTS parses (
  sha256 TEXT NOT NULL,
  parser TEXT NOT NULL,
  page_no INTEGER NOT NULL,
  text TEXT NOT NULL,
  page_status TEXT NOT NULL,
  PRIMARY KEY (sha256, parser, page_no)
);

CREATE TABLE IF NOT EXISTS generations (
  library_id TEXT NOT NULL,
  gen INTEGER NOT NULL,
  state TEXT NOT NULL,
  embed_sig TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (library_id, gen)
);

CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  document_id TEXT NOT NULL,
  gen INTEGER NOT NULL,
  page_no INTEGER NOT NULL,
  ordinal INTEGER NOT NULL,
  text TEXT NOT NULL,
  sha256 TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS embeddings (
  chunk_id TEXT PRIMARY KEY,
  dim INTEGER NOT NULL,
  sig TEXT NOT NULL,
  vec BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS materials (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  document_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  url TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS annotations (
  id TEXT PRIMARY KEY,
  material_id TEXT NOT NULL,
  quote TEXT NOT NULL,
  note TEXT NOT NULL,
  page_no INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS paths (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL,
  mode TEXT NOT NULL DEFAULT 'outline',
  teaching TEXT NOT NULL DEFAULT 'probe_first',
  prior_knowledge TEXT NOT NULL DEFAULT '',
  target_level TEXT NOT NULL DEFAULT '',
  time_budget TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS path_leases (
  path_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  until_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS modules (
  id TEXT PRIMARY KEY,
  path_id TEXT NOT NULL,
  name TEXT NOT NULL,
  objective TEXT NOT NULL DEFAULT '',
  ord INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS points (
  id TEXT PRIMARY KEY,
  path_id TEXT NOT NULL,
  module_id TEXT NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  ord INTEGER NOT NULL,
  mastery REAL NOT NULL DEFAULT 0,
  qualitative INTEGER NOT NULL DEFAULT 0,
  mastery_source TEXT NOT NULL DEFAULT '',
  attempts_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS repetitions (
  point_id TEXT PRIMARY KEY,
  path_id TEXT NOT NULL,
  interval_index INTEGER NOT NULL DEFAULT 0,
  next_review_at TEXT NOT NULL,
  consecutive_correct INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS interactions (
  id TEXT PRIMARY KEY,
  path_id TEXT NOT NULL,
  point_id TEXT NOT NULL,
  session_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  question_type TEXT NOT NULL,
  prompt TEXT NOT NULL,
  expected_answer TEXT NOT NULL,
  options_json TEXT NOT NULL DEFAULT '[]',
  explanation TEXT NOT NULL DEFAULT '',
  user_answer TEXT NOT NULL DEFAULT '',
  score REAL NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS questions (
  id TEXT PRIMARY KEY,
  stem TEXT NOT NULL,
  qtype TEXT NOT NULL,
  options_json TEXT NOT NULL DEFAULT '[]',
  answer TEXT NOT NULL,
  explanation TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  source_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS practice_state (
  question_id TEXT PRIMARY KEY,
  due_at TEXT NOT NULL,
  interval_days REAL NOT NULL,
  ease REAL NOT NULL,
  lapses INTEGER NOT NULL DEFAULT 0,
  review_count INTEGER NOT NULL DEFAULT 0,
  streak INTEGER NOT NULL DEFAULT 0,
  suspended INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS practice_events (
  id TEXT PRIMARY KEY,
  question_id TEXT NOT NULL,
  rating TEXT NOT NULL,
  at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS journal (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  topic TEXT NOT NULL DEFAULT '',
  why TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  next_focus TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS journal_records (
  id TEXT PRIMARY KEY,
  insight TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS books (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  depth TEXT NOT NULL DEFAULT 'standard',
  proposal TEXT NOT NULL DEFAULT '',
  spine_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS book_blocks (
  id TEXT PRIMARY KEY,
  book_id TEXT NOT NULL,
  ord INTEGER NOT NULL,
  kind TEXT NOT NULL,
  body TEXT NOT NULL,
  page_no INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS book_progress (
  book_id TEXT NOT NULL,
  reader TEXT NOT NULL DEFAULT 'local',
  page_no INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (book_id, reader)
);

CREATE TABLE IF NOT EXISTS index_journal (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  document_id TEXT NOT NULL,
  gen INTEGER NOT NULL,
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT ''
);
`

const ftsSchema = `
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
  text,
  chunk_id UNINDEXED,
  library_id UNINDEXED,
  gen UNINDEXED,
  tokenize = 'unicode61 remove_diacritics 2'
);
`

// Open 打开 root/learn.sqlite，建表，并把 pagesRoot 里尚未迁入的 Markdown 抄进笔记。
// root 是学习目录。pagesRoot 是旧的 pages 目录，可以不存在。
func Open(root, pagesRoot string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "raw"), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(filepath.Join(root, "learn.sqlite")) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, root: root, index: map[string]*indexJob{}, vectors: map[string]*vectorSet{}, closing: make(chan struct{})}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(ftsSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("fts5: %w", err)
	}
	// 建表语句停在第一版。之后的列和表都由迁移补上，新库和旧库走同一条路。
	if err := s.runMigrations(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.migratePages(pagesRoot); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.ensureNotesLibrary(); err != nil {
		_ = db.Close()
		return nil, err
	}
	_, _ = db.Exec(`INSERT INTO journal (id) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM journal WHERE id = 1)`)
	// 上次退出时还在后台跑的活不会自己接上，各个域在这里把状态收拾成人能继续的样子。
	s.recoverLibraries()
	s.recoverBooks()
	return s, nil
}

// Close 先让后台的索引和写书收工并等它们退出，再关闭数据库。root 下的原文文件留着。
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	if s.closing != nil {
		s.closeOnce.Do(func() { close(s.closing) })
		s.bg.Wait()
	}
	return s.db.Close()
}

// Root 返回学习目录。
func (s *Store) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}
