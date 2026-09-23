package video

import (
	"bytes"
	"encoding/json"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		hash := strings.TrimPrefix(r.URL.Path, "/media/")
		hash = strings.TrimSpace(strings.Split(hash, "?")[0])
		if hash == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		if path, err := e.ServeFile(hash); err == nil {
			f, err := os.Open(path)
			if err == nil {
				defer f.Close()
				st, _ := f.Stat()
				if ct := mediaType(hash, nil); ct != "" {
					w.Header().Set("Content-Type", ct)
				}
				http.ServeContent(w, r, hash, st.ModTime(), f)
				return
			}
		}
		raw, err := e.GetBytes(hash)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if ct := mediaType(hash, raw); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		http.ServeContent(w, r, hash, time.Time{}, bytes.NewReader(raw))
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return nil
}

func mediaType(hash string, raw []byte) string {
	if len(raw) > 12 && string(raw[4:8]) == "ftyp" {
		return "video/mp4"
	}
	if len(raw) >= 3 && raw[0] == 0xff && raw[1] == 0xd8 {
		return "image/jpeg"
	}
	if len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	_ = hash
	if len(raw) > 0 {
		return http.DetectContentType(raw)
	}
	return ""
}

type Status struct {
	MediaBase string `json:"media_base"`
	FFmpeg    bool   `json:"ffmpeg"`
	FFmpegBin string `json:"ffmpeg_bin"`
	FFProbe   string `json:"ffprobe_bin"`
	DB        string `json:"db"`
}

func (e *Engine) Status() Status {
	return Status{
		MediaBase: e.MediaBase,
		FFmpeg:    e.FFmpegOK(),
		FFmpegBin: e.FFMPEG,
		FFProbe:   e.FFProbe,
		DB:        e.Dir,
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
		"dir":              st.DB,
		"dramas":           n,
		"content_language": e.ContentLanguage(),
		"modes":            Modes(),
		"missing":          e.missingProviderKinds(),
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
