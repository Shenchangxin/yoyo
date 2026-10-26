// Package video 是视频数据与渲染。短片、短剧、画布和剪辑的文档与成片由这个包保管，画面和声音由本包的 ffmpeg、Remotion 与供应商任务渲染。模型循环仍是 runtime.Run。
package video

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/filestore"
	"github.com/Shenchangxin/yoyo/internal/vault"
	"github.com/Shenchangxin/yoyo/internal/video/protocol"
)

// Secrets 按名字保存和取出供应商密钥。
type Secrets interface {
	// Lease 按名字取出密钥。
	Lease(name string) (string, error)
	// Set 保存一把密钥。
	Set(name, value string)
	// Get 读取一把密钥。
	Get(name string) (string, error)
}

// Blobs 按标识存放和取回原始字节。
type Blobs interface {
	// PutRaw 写入原始字节并返回标识。
	PutRaw([]byte) (string, error)
	// GetRaw 按标识读回字节。
	GetRaw(hash string) ([]byte, error)
	// Path 返回该标识对应的本地路径。
	Path(hash string) string
}

// HTTPDoer 发送一次 HTTP 请求。
type HTTPDoer interface {
	// Do 执行请求并返回响应。
	Do(*http.Request) (*http.Response, error)
}

// Engine 打开视频目录，保管文档、连接、素材标识和渲染用的外部程序路径。
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
	mediaSrv  *http.Server

	mu       sync.Mutex
	binds    map[string]Bind
	inflight sync.Map
	uploads  sync.Map

	filmMu      sync.Mutex
	filmPending []string
	filmRunning bool
	ctx         context.Context
	cancel      context.CancelFunc
	OnJob       func(Job)
	OnHub       func(kind string, payload map[string]any)
	Agent       *CanvasAgentBridge
	agents      canvasAgentBook

	castFile string
	castURL  string
	castSHA  *string
	castHTTP HTTPDoer

	Proto      *protocol.Registry
	PluginDir  string
	WHISPER    string
	YTDLP      string
	StampBin   string
	RegionBin  string
	BrowserBin string

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

	rmMu      sync.Mutex
	rmPhase   string
	rmPercent int
	rmErr     string
	rmWait    chan struct{}
	rmNode    string
	rmNPM     string
	rmBin     string

	trendCache map[string]trendCacheEntry
}

// Bind 记录会话对应的集和短剧。
type Bind struct {
	SessionID string `json:"session_id"`
	EpisodeID string `json:"episode_id"`
	DramaID   string `json:"drama_id"`
}

// Open 用给定目录、字节库和密钥库打开引擎，连接表由 OpenWith 自己创建。
func Open(dir string, cas Blobs, secrets Secrets) (*Engine, error) {
	return OpenWith(dir, cas, secrets, nil)
}

// OpenWith 创建目录和 docs 文档库。conn 为空时在目录下的 connections 打开连接表。默认 HTTP 超时 12 分钟。随后在 bin 里找 ffmpeg 和 whisper，并查找 yt-dlp、whisperx、区域检测和浏览器。接着迁移旧的 sqlite，播种样式，装入画布插件和短片素材，恢复未完成任务，再按 800 毫秒节拍推进排队中的生成任务。
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
	e.lookFilmTools()
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
	e.ctx = ctx
	e.cancel = cancel
	e.resumeJobs()
	go e.loop(ctx)
	return e, nil
}

// OpenDefault 在密钥库非空时，用视频目录的上一级打开连接表，再交给 OpenWith。
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

// Close 在 1 秒内关掉媒体服务，并取消引擎上下文。
func (e *Engine) Close() error {
	if e.mediaSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = e.mediaSrv.Shutdown(ctx)
		cancel()
		e.mediaSrv = nil
	}
	if e.cancel != nil {
		e.cancel()
	}
	return nil
}

// Setting 读键值设置。缺了或是空串时用 fallback。
func (e *Engine) Setting(key, fallback string) string {
	m := e.kvSettings()
	if v := m[key]; v != "" {
		return v
	}
	return fallback
}

// SetSetting 改一个键并写回设置文档。
func (e *Engine) SetSetting(key, value string) error {
	m := e.kvSettings()
	m[key] = value
	return e.putJSON(fileSettings, m)
}

// ContentLanguage 读 content_language，缺省是 zh。
func (e *Engine) ContentLanguage() string {
	return e.Setting("content_language", "zh")
}

// emit 在 OnJob 非空时把任务交出去。
func (e *Engine) emit(j Job) {
	if e.OnJob != nil {
		e.OnJob(j)
	}
}

// BindSession 把会话绑到已存在的一集，短剧标识取自该集。
func (e *Engine) BindSession(sessionID, episodeID string) error {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return err
	}
	return e.putBind(Bind{SessionID: sessionID, EpisodeID: episodeID, DramaID: ep.DramaID})
}

// BindSessionDrama 把会话绑到已存在的短剧。当前集仍属于这部剧时，集标识保留。
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

// putBind 把绑定写入文档并放进内存表。
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

// ReleaseWorkshopBinds 去掉该会话的短片、短剧和画布绑定，使会话回到剪辑工具和 video_route 那条通道。
func (e *Engine) ReleaseWorkshopBinds(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	sm := e.loadStudioBinds()
	if _, ok := sm[sessionID]; ok {
		delete(sm, sessionID)
		_ = e.saveStudioBinds(sm)
	}
	cm := e.loadCanvasBinds()
	if _, ok := cm[sessionID]; ok {
		delete(cm, sessionID)
		_ = e.saveCanvasBinds(cm)
	}
	bm := e.loadBinds()
	if _, ok := bm[sessionID]; ok {
		delete(bm, sessionID)
		_ = e.saveBinds(bm)
		e.mu.Lock()
		delete(e.binds, sessionID)
		e.mu.Unlock()
	}
}

// SessionBind 先查内存，没有再读文档，并把读到的绑定放回内存。
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

// leaseProvider 从密钥库取出该供应商的 VaultKey。密钥库为空时返回 vault missing。
func (e *Engine) leaseProvider(p Provider) (string, error) {
	if e.Vault == nil {
		return "", fmt.Errorf("vault missing")
	}
	return e.Vault.Lease(p.VaultKey)
}

// marshalJSON 把值编成 JSON 文本。失败时返回 {}。
func marshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
