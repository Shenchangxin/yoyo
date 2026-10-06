package video

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/filestore"
	"github.com/Shenchangxin/yoyo/internal/vault"
	"github.com/Shenchangxin/yoyo/internal/video/protocol"
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
	docs      *filestore.Store
	Conn      *connection.Registry
	Dir       string
	CAS       Blobs
	Vault     Secrets
	HTTP      HTTPDoer
	FFMPEG    string
	FFProbe   string
	MediaBase string

	mu       sync.Mutex
	binds    map[string]Bind
	inflight sync.Map
	uploads  sync.Map
	cancel   context.CancelFunc
	OnJob    func(Job)
	OnHub    func(kind string, payload map[string]any)

	Proto     *protocol.Registry
	PluginDir string
	WHISPER   string

	ffmpegHTTP HTTPDoer
	ffmpegOS   string
	ffmpegArch string
	ffMu       sync.Mutex
	ffPhase    string
	ffPercent  int
	ffBytes    int64
	ffTotal    int64
	ffArchive  string
	ffErr      string
	ffWait     chan struct{}
}

type Bind struct {
	SessionID string `json:"session_id"`
	EpisodeID string `json:"episode_id"`
	DramaID   string `json:"drama_id"`
}

func Open(dir string, cas Blobs, secrets Secrets) (*Engine, error) {
	return OpenWith(dir, cas, secrets, nil)
}

func OpenWith(dir string, cas Blobs, secrets Secrets, conn *connection.Registry) (*Engine, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	docs, err := filestore.New(filepath.Join(dir, "docs"))
	if err != nil {
		return nil, err
	}
	if conn == nil {
		conn, err = connection.Open(filepath.Join(dir, "connections"), secrets)
		if err != nil {
			return nil, err
		}
	}
	e := &Engine{
		docs:  docs,
		Conn:  conn,
		Dir:   dir,
		CAS:   cas,
		Vault: secrets,
		HTTP:  &http.Client{Timeout: 12 * time.Minute},
		binds: map[string]Bind{},
	}
	e.FFMPEG, e.FFProbe = LookFFmpegIn([]string{filepath.Join(dir, "bin")})
	e.WHISPER = LookWhisper()
	if err := migrateSQLiteIfPresent(e); err != nil {
		return nil, err
	}
	if err := e.seedStyles(); err != nil {
		return nil, err
	}
	e.PluginDir = findYingcePluginDir(dir)
	_ = e.loadCanvasPlugins()
	_ = e.seedStudioAssets()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.resumeJobs()
	go e.loop(ctx)
	return e, nil
}

func OpenDefault(dir string, cas *artifact.Store, v *vault.Store) (*Engine, error) {
	var conn *connection.Registry
	if v != nil {
		home := filepath.Dir(dir)
		c, err := connection.Open(filepath.Join(home, "connections"), v)
		if err != nil {
			return nil, err
		}
		conn = c
	}
	return OpenWith(dir, cas, v, conn)
}

func (e *Engine) Close() error {
	if e.cancel != nil {
		e.cancel()
	}
	return nil
}

func (e *Engine) Setting(key, fallback string) string {
	m := e.kvSettings()
	if v := m[key]; v != "" {
		return v
	}
	return fallback
}

func (e *Engine) SetSetting(key, value string) error {
	m := e.kvSettings()
	m[key] = value
	return e.putJSON(fileSettings, m)
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
	return e.putBind(Bind{SessionID: sessionID, EpisodeID: episodeID, DramaID: ep.DramaID})
}

func (e *Engine) BindSessionDrama(sessionID, dramaID string) error {
	d, err := e.GetDrama(dramaID)
	if err != nil {
		return err
	}
	b := Bind{SessionID: sessionID, DramaID: d.ID}
	if cur, ok := e.SessionBind(sessionID); ok && cur.EpisodeID != "" {
		if ep, err := e.GetEpisode(cur.EpisodeID); err == nil && ep.DramaID == d.ID {
			b.EpisodeID = ep.ID
		}
	}
	return e.putBind(b)
}

func (e *Engine) putBind(b Bind) error {
	m := e.loadBinds()
	m[b.SessionID] = b
	if err := e.saveBinds(m); err != nil {
		return err
	}
	e.mu.Lock()
	e.binds[b.SessionID] = b
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
	m := e.loadBinds()
	b, ok = m[sessionID]
	if !ok {
		return Bind{}, false
	}
	e.mu.Lock()
	e.binds[sessionID] = b
	e.mu.Unlock()
	return b, true
}

func (e *Engine) leaseProvider(p Provider) (string, error) {
	if e.Vault == nil {
		return "", fmt.Errorf("vault missing")
	}
	return e.Vault.Lease(p.VaultKey)
}

func marshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
