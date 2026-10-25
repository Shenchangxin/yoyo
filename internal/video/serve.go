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

// MediaAssetMiddleware serves /media and /resource on the app origin.
// The loopback media server is http, and an https webview will not play it.
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

func (e *Engine) mediaHandler() http.Handler {
	mux := http.NewServeMux()
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

func peekMedia(w http.ResponseWriter, f *os.File) bool {
	head, err := peekOpen(f)
	if err != nil {
		return false
	}
	applyMediaType(w, head)
	return true
}

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

func headBytes(raw []byte) []byte {
	if len(raw) > 512 {
		return raw[:512]
	}
	return raw
}

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

func applyMediaType(w http.ResponseWriter, raw []byte) {
	ct := mediaType(raw)
	if ct == "" {
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// CAS objects have no filename. Go's sniffer calls an MP4
// application/octet-stream, and the webview then paints a black 0s player.
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

type Status struct {
	MediaBase     string         `json:"media_base"`
	FFmpeg        bool           `json:"ffmpeg"`
	FFmpegBin     string         `json:"ffmpeg_bin"`
	FFProbe       string         `json:"ffprobe_bin"`
	FFmpegInstall map[string]any `json:"ffmpeg_install"`
	DB            string         `json:"db"`
}

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

func (e *Engine) JSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

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
