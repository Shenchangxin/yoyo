package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/vault"
	_ "modernc.org/sqlite"
)

type Secrets interface {
	Lease(name string) (string, error)
	Set(name, value string)
	Get(name string) (string, error)
}

type Blobs interface {
	PutRaw([]byte) (string, error)
	GetRaw(hash string) ([]byte, error)
	Path(hash string) string
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Engine struct {
	DB     *sql.DB
	Dir    string
	CAS    Blobs
	Vault  Secrets
	HTTP   HTTPDoer
	FFMPEG string
	FFProbe string
	MediaBase string

	mu     sync.Mutex
	binds  map[string]Bind
	cancel context.CancelFunc
	OnJob  func(Job)
}

type Bind struct {
	SessionID string `json:"session_id"`
	EpisodeID string `json:"episode_id"`
	DramaID   string `json:"drama_id"`
}

func Open(dir string, cas Blobs, secrets Secrets) (*Engine, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "video.sqlite")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}
	e := &Engine{
		DB:    db,
		Dir:   dir,
		CAS:   cas,
		Vault: secrets,
		HTTP:  &http.Client{Timeout: 12 * time.Minute},
		binds: map[string]Bind{},
	}
	e.FFMPEG, e.FFProbe = LookFFmpeg()
	if err := e.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := e.seedStyles(); err != nil {
		_ = db.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.resumeJobs()
	go e.loop(ctx)
	return e, nil
}

func OpenDefault(dir string, cas *artifact.Store, v *vault.Store) (*Engine, error) {
	return Open(dir, cas, v)
}

func (e *Engine) Close() error {
	if e.cancel != nil {
		e.cancel()
	}
	if e.DB != nil {
		return e.DB.Close()
	}
	return nil
}

func (e *Engine) migrate() error {
	if _, err := e.DB.Exec(schemaV1); err != nil {
		return err
	}
	_, err := e.DB.Exec(`INSERT OR IGNORE INTO schema_migrations(version) VALUES (1)`)
	return err
}

func (e *Engine) Setting(key, fallback string) string {
	var v string
	err := e.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil || v == "" {
		return fallback
	}
	return v
}

func (e *Engine) SetSetting(key, value string) error {
	_, err := e.DB.Exec(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (e *Engine) ContentLanguage() string {
	return e.Setting("content_language", "zh")
}

func (e *Engine) emit(j Job) {
	if e.OnJob != nil {
		e.OnJob(j)
	}
}

func (e *Engine) BindSession(sessionID, episodeID string) error {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return err
	}
	now := Now()
	_, err = e.DB.Exec(`INSERT INTO session_binds(session_id, episode_id, drama_id, updated_at) VALUES(?,?,?,?)
		ON CONFLICT(session_id) DO UPDATE SET episode_id=excluded.episode_id, drama_id=excluded.drama_id, updated_at=excluded.updated_at`,
		sessionID, episodeID, ep.DramaID, now)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.binds[sessionID] = Bind{SessionID: sessionID, EpisodeID: episodeID, DramaID: ep.DramaID}
	e.mu.Unlock()
	return nil
}

func (e *Engine) SessionBind(sessionID string) (Bind, bool) {
	e.mu.Lock()
	b, ok := e.binds[sessionID]
	e.mu.Unlock()
	if ok {
		return b, true
	}
	var out Bind
	err := e.DB.QueryRow(`SELECT session_id, episode_id, drama_id FROM session_binds WHERE session_id = ?`, sessionID).
		Scan(&out.SessionID, &out.EpisodeID, &out.DramaID)
	if err != nil {
		return Bind{}, false
	}
	e.mu.Lock()
	e.binds[sessionID] = out
	e.mu.Unlock()
	return out, true
}

func (e *Engine) leaseProvider(p Provider) (string, error) {
	if e.Vault == nil {
		return "", fmt.Errorf("vault missing")
	}
	return e.Vault.Lease(p.VaultKey)
}

func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		return "{}"
	}
	return string(b)
}
