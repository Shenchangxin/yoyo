// serve.go 在本机环回地址和应用程序源上提供媒体字节。

package video

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// StartMediaServer 在 127.0.0.1 的随机端口监听，把地址记到 MediaBase，并在后台 Serve。
func (e *Engine) StartMediaServer() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	e.MediaBase = "http://" + ln.Addr().String()
	srv := &http.Server{Handler: e.mediaHandler()}
	e.mediaSrv = srv
	go func() { _ = srv.Serve(ln) }()
	return nil
}

// MediaAssetMiddleware 在应用自己的源上提供 /media 和 /resource。环回媒体服务是 http，https 网页视图不会播放它。
func MediaAssetMiddleware(e *Engine, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if e == nil {
		return next
	}
	h := e.mediaHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/media/") || strings.HasPrefix(p, "/resource/") {
			h.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// mediaHandler 处理 /resource/ 和 /media/。允许任意源，并放行 Range；OPTIONS 返回 204。资源按标识交给 ServeResourceFile，媒体按哈希走本地文件或 GetBytes。缓存一年且 immutable。
func (e *Engine) mediaHandler() http.Handler {
	mux := http.NewServeMux()
	// cors 写上允许任意源和 Range。OPTIONS 返回 204 并让调用方停止后续处理。
	cors := func(w http.ResponseWriter, r *http.Request) bool {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return true
		}
		return false
	}
	mux.HandleFunc("/resource/", func(w http.ResponseWriter, r *http.Request) {
		if cors(w, r) {
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/resource/")
		id = strings.TrimSpace(strings.Split(id, "?")[0])
		if id == "" {
			http.NotFound(w, r)
			return
		}
		path, raw, mime, err := e.ServeResourceFile(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		if strings.EqualFold(mime, "application/octet-stream") {
			mime = ""
		}
		if mime != "" {
			w.Header().Set("Content-Type", mime)
			w.Header().Set("X-Content-Type-Options", "nosniff")
		} else {
			applyMediaType(w, headBytes(raw))
		}
		if path != "" {
			if f, err := os.Open(path); err == nil {
				defer f.Close()
				st, sterr := f.Stat()
				if sterr == nil && (mime != "" || peekMedia(w, f)) {
					http.ServeContent(w, r, id, st.ModTime(), f)
					return
				}
			}
		}
		http.ServeContent(w, r, id, time.Time{}, bytes.NewReader(raw))
	})
	mux.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
		if cors(w, r) {
			return
		}
		hash := strings.TrimPrefix(r.URL.Path, "/media/")
		hash = strings.TrimSpace(strings.Split(hash, "?")[0])
		if !safeCASHash(hash) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		if path, err := e.ServeFile(hash); err == nil {
			if f, err := os.Open(path); err == nil {
				defer f.Close()
				st, sterr := f.Stat()
				if sterr == nil && peekMedia(w, f) {
					http.ServeContent(w, r, hash, st.ModTime(), f)
					return
				}
			}
		}
		raw, err := e.GetBytes(hash)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		applyMediaType(w, headBytes(raw))
		http.ServeContent(w, r, hash, time.Time{}, bytes.NewReader(raw))
	})
	return mux
}

// peekMedia 读文件头并写上媒体类型。读失败时返回假。
func peekMedia(w http.ResponseWriter, f *os.File) bool {
	head, err := peekOpen(f)
	if err != nil {
		return false
	}
	applyMediaType(w, head)
	return true
}

// peekOpen 读最多 512 字节，再把文件指针拨回开头。
func peekOpen(f *os.File) ([]byte, error) {
	head := make([]byte, 512)
	n, err := f.Read(head)
	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		return nil, seekErr
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return head[:n], nil
}

// headBytes 返回正文的前 512 字节；更短时返回全部。
func headBytes(raw []byte) []byte {
	if len(raw) > 512 {
		return raw[:512]
	}
	return raw
}

// presentableMime 保留已经具体的 MIME。空串或 application/octet-stream 时改用 mediaType，仍没有且原文非空时再用 http.DetectContentType。
func presentableMime(mime string, raw []byte) string {
	mime = strings.TrimSpace(mime)
	if mime != "" && !strings.EqualFold(mime, "application/octet-stream") {
		return mime
	}
	if sniffed := mediaType(raw); sniffed != "" {
		return sniffed
	}
	if mime == "" && len(raw) > 0 {
		if ct := http.DetectContentType(raw); ct != "" {
			return ct
		}
	}
	return mime
}

// applyMediaType 在 mediaType 认得出类型时写上 Content-Type 和 nosniff。
func applyMediaType(w http.ResponseWriter, raw []byte) {
	ct := mediaType(raw)
	if ct == "" {
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// mediaType 识别没有文件名的内容寻址对象。ftyp 视为 video/mp4，EBML 头视为 video/webm，RIFF AVI 视为 video/x-msvideo，并识别 JPEG 和 PNG。嗅探结果为空或 application/octet-stream 时返回空串，避免网页视图把 MP4 画成 0 秒黑屏。
func mediaType(raw []byte) string {
	if len(raw) >= 12 && string(raw[4:8]) == "ftyp" {
		return "video/mp4"
	}
	if len(raw) >= 4 && raw[0] == 0x1a && raw[1] == 0x45 && raw[2] == 0xdf && raw[3] == 0xa3 {
		return "video/webm"
	}
	if len(raw) >= 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "AVI " {
		return "video/x-msvideo"
	}
	if len(raw) >= 3 && raw[0] == 0xff && raw[1] == 0xd8 {
		return "image/jpeg"
	}
	if len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(raw) == 0 {
		return ""
	}
	ct := http.DetectContentType(raw)
	if ct == "" || ct == "application/octet-stream" {
		return ""
	}
	return ct
}

// Status 是媒体基址、ffmpeg 是否可用、安装进度和视频目录。
type Status struct {
	MediaBase     string         `json:"media_base"`
	FFmpeg        bool           `json:"ffmpeg"`
	FFmpegBin     string         `json:"ffmpeg_bin"`
	FFProbe       string         `json:"ffprobe_bin"`
	FFmpegInstall map[string]any `json:"ffmpeg_install"`
	DB            string         `json:"db"`
}

// Status 刷新 ffmpeg 路径后返回当前媒体服务和安装状态。DB 字段填的是视频目录。
func (e *Engine) Status() Status {
	e.refreshFFmpeg()
	return Status{
		MediaBase:     e.MediaBase,
		FFmpeg:        e.FFmpegOK(),
		FFmpegBin:     e.FFMPEG,
		FFProbe:       e.FFProbe,
		FFmpegInstall: e.FFmpegInstallSnapshot(),
		DB:            e.Dir,
	}
}

// JSON 把值编成 RawMessage。编码错误被忽略。
func (e *Engine) JSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// Snapshot 汇总媒体地址、ffmpeg、短剧数量、内容语言、工作台、缺密钥的供应商种类，以及 Remotion、whisper、yt-dlp、whisperx、区域检测和浏览器是否有路径。
func (e *Engine) Snapshot() map[string]any {
	st := e.Status()
	n := 0
	if list, err := e.ListDramas(); err == nil {
		n = len(list)
	}
	return map[string]any{
		"media_base":       st.MediaBase,
		"ffmpeg":           st.FFmpeg,
		"ffmpeg_bin":       st.FFmpegBin,
		"ffprobe_bin":      st.FFProbe,
		"ffmpeg_install":   st.FFmpegInstall,
		"dir":              st.DB,
		"dramas":           n,
		"content_language": e.ContentLanguage(),
		"modes":            Modes(),
		"missing":          e.missingProviderKinds(),
		"remotion":         e.RemotionSnapshot(),
		"whisper":          strings.TrimSpace(e.WHISPER) != "",
		"whisper_bin":      e.WHISPER,
		"ytdlp":            strings.TrimSpace(e.YTDLP) != "",
		"ytdlp_bin":        e.YTDLP,
		"stamp":            strings.TrimSpace(e.StampBin) != "",
		"stamp_bin":        e.StampBin,
		"region":           strings.TrimSpace(e.RegionBin) != "",
		"region_bin":       e.RegionBin,
		"browser":          strings.TrimSpace(e.BrowserBin) != "",
		"browser_bin":      e.BrowserBin,
	}
}

// missingProviderKinds 列出 image 和 video 里没有一条既启用又有密钥的供应商种类。两种都有时返回空切片。
func (e *Engine) missingProviderKinds() []string {
	var out []string
	for _, kind := range []string{"image", "video"} {
		list, _ := e.ListProviders(kind)
		ok := false
		for _, p := range list {
			if p.IsActive && p.HasKey {
				ok = true
				break
			}
		}
		if !ok {
			out = append(out, kind)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}
