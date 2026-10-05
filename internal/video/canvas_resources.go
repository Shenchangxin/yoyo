package video

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type canvasResource struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	Provider       string `json:"provider"`
	Endpoint       string `json:"endpoint"`
	Bucket         string `json:"bucket"`
	ObjectKey      string `json:"objectKey"`
	PublicURL      string `json:"publicUrl"`
	MimeType       string `json:"mimeType"`
	Size           int64  `json:"size"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	DurationMs     int    `json:"durationMs,omitempty"`
	PlaybackStatus string `json:"playbackStatus,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	CASHash        string `json:"-"`
	PosterHash     string `json:"-"`
}

type canvasUpload struct {
	ID         string
	Kind       string
	FileName   string
	Mime       string
	Size       int
	ChunkSize  int
	ChunkCount int
	Width      int
	Height     int
	DurationMs int
	Dir        string
	mu         sync.Mutex
	got        map[int]bool
}

func (e *Engine) resourcePublicURL(id, hash string) string {
	if id == "" {
		return e.MediaURL(hash)
	}
	if e.MediaBase != "" {
		return strings.TrimRight(e.MediaBase, "/") + "/resource/" + id
	}
	return "/resource/" + id
}

func (e *Engine) canvasResourceFromRow(id, hash, kind, mime string, bytes int64, width, height, duration int, poster, playback, fileName, created, updated string) canvasResource {
	if kind == "" {
		kind = "file"
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return canvasResource{
		ID: id, UserID: "local", Kind: kind, Status: "ready", Provider: "cas",
		Endpoint: "local", Bucket: "cas", ObjectKey: hash,
		PublicURL: e.resourcePublicURL(id, hash), MimeType: mime, Size: bytes,
		Width: width, Height: height, DurationMs: duration, PlaybackStatus: playback,
		CreatedAt: created, UpdatedAt: updated, CASHash: hash, PosterHash: poster,
	}
}

func (e *Engine) getCanvasResource(id string) (canvasResource, error) {
	rec, err := getDoc[canvasResourceRec](e, colResources, id)
	if err != nil || rec.DeletedAt != "" {
		return canvasResource{}, fmt.Errorf("resource not found")
	}
	return e.canvasResourceFromRow(rec.ID, rec.CASHash, rec.Kind, rec.Mime, rec.Bytes, rec.Width, rec.Height, rec.DurationMs, rec.PosterHash, rec.PlaybackStatus, rec.FileName, rec.CreatedAt, rec.UpdatedAt), nil
}

func (e *Engine) putCanvasBytes(kind, mime, fileName string, raw []byte, width, height, duration int, id string) (canvasResource, error) {
	if len(raw) == 0 {
		return canvasResource{}, fmt.Errorf("empty file")
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return canvasResource{}, err
	}
	if mime == "" {
		mime = http.DetectContentType(raw)
	}
	if kind == "" {
		if strings.HasPrefix(mime, "image/") {
			kind = "image"
		} else if strings.HasPrefix(mime, "video/") {
			kind = "video"
		} else if strings.HasPrefix(mime, "audio/") {
			kind = "audio"
		} else {
			kind = "file"
		}
	}
	if id == "" {
		id = NewID()
	}
	now := Now()
	poster := ""
	if kind == "image" {
		if th, err := e.Thumb(hash); err == nil {
			poster = th
		}
	}
	if kind == "video" {
		if b, err := e.Poster(e.FilePath(hash)); err == nil {
			if ph, err := e.PutBytes(b); err == nil {
				poster = ph
			}
		}
	}
	rec := canvasResourceRec{
		ID: id, CASHash: hash, Kind: kind, Mime: mime, Bytes: int64(len(raw)),
		Width: width, Height: height, DurationMs: duration, PosterHash: poster,
		PlaybackStatus: "none", FileName: fileName, CreatedAt: now, UpdatedAt: now,
	}
	if cur, err := getDoc[canvasResourceRec](e, colResources, id); err == nil {
		rec.CreatedAt = cur.CreatedAt
	}
	if err := e.putDoc(colResources, id, rec); err != nil {
		return canvasResource{}, err
	}
	return e.getCanvasResource(id)
}

func (e *Engine) putCanvasResourceFromHash(hash, kind, mime string, bytes int64, width, height, duration int) (canvasResource, error) {
	if hash == "" {
		return canvasResource{}, fmt.Errorf("cas hash missing")
	}
	id := NewID()
	now := Now()
	rec := canvasResourceRec{
		ID: id, CASHash: hash, Kind: kind, Mime: mime, Bytes: bytes,
		Width: width, Height: height, DurationMs: duration, PosterHash: "",
		PlaybackStatus: "none", CreatedAt: now, UpdatedAt: now,
	}
	if err := e.putDoc(colResources, id, rec); err != nil {
		return canvasResource{}, err
	}
	return e.getCanvasResource(id)
}

func (e *Engine) finishCanvasMedia(j *Job, raw []byte, video bool) error {
	kind := "image"
	if video {
		kind = "video"
	}
	p := parseParams(j.Params)
	if p.Mode == "audio" {
		kind = "audio"
	}
	if p.Mode == "video" {
		kind = "video"
	}
	if p.Mode == "image" {
		kind = "image"
	}
	res, err := e.putCanvasResourceFromHash(j.ResultHash, kind, "", int64(len(raw)), 0, 0, 0)
	if err != nil {
		p.ResultState = "FAILED_RETRYABLE"
		p.MediaStage = "register"
		j.Params = marshalJSON(p)
		return err
	}
	p.ResourceID = res.ID
	p.ResultState = "READY"
	p.MediaStage = "completed"
	j.Params = marshalJSON(p)
	return nil
}

func (e *Engine) resourceAccess(id, purpose, variant string) (map[string]any, error) {
	res, err := e.getCanvasResource(id)
	if err != nil {
		return nil, err
	}
	url := res.PublicURL
	if variant == "playback" && res.PosterHash != "" && purpose == "display" {
		// original CAS playback; media server serves original bytes
	}
	_ = purpose
	now := Now()
	return map[string]any{
		"resourceId":       id,
		"requestedVariant": variant,
		"actualVariant":    "original",
		"url":              url,
		"delivery":         "platform-local",
		"issuedAt":         now,
		"refreshAt":        now,
		"revision":         res.CASHash,
	}, nil
}

func (e *Engine) storageUsage() map[string]any {
	var used int64
	for _, r := range loadCol[canvasResourceRec](e, colResources) {
		if r.DeletedAt == "" {
			used += r.Bytes
		}
	}
	return map[string]any{"usedBytes": used, "totalBytes": int64(1 << 40)}
}

func (e *Engine) startChunkUpload(kind, fileName string, size, width, height, duration int) (*canvasUpload, error) {
	const chunk = 8 << 20
	if size <= 0 {
		return nil, fmt.Errorf("invalid size")
	}
	count := (size + chunk - 1) / chunk
	id := NewID()
	dir := filepath.Join(e.Dir, "uploads", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	now := Now()
	up := canvasUploadRec{
		ID: id, Kind: kind, FileName: fileName, Size: size, ChunkSize: chunk, ChunkCount: count,
		Width: width, Height: height, DurationMs: duration, Received: 0, Dir: dir, CreatedAt: now,
	}
	if err := e.putDoc(colUploads, id, up); err != nil {
		return nil, err
	}
	u := &canvasUpload{ID: id, Kind: kind, FileName: fileName, Size: size, ChunkSize: chunk, ChunkCount: count, Width: width, Height: height, DurationMs: duration, Dir: dir, got: map[int]bool{}}
	e.uploads.Store(id, u)
	return u, nil
}

func (e *Engine) putChunk(id string, index int, raw []byte) error {
	v, ok := e.uploads.Load(id)
	if !ok {
		return fmt.Errorf("upload session expired")
	}
	u := v.(*canvasUpload)
	if index < 0 || index >= u.ChunkCount {
		return fmt.Errorf("chunk index out of range")
	}
	path := filepath.Join(u.Dir, fmt.Sprintf("%06d", index))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return err
	}
	u.mu.Lock()
	u.got[index] = true
	n := len(u.got)
	u.mu.Unlock()
	if rec, err := getDoc[canvasUploadRec](e, colUploads, id); err == nil {
		rec.Received = n
		_ = e.putDoc(colUploads, id, rec)
	}
	return nil
}

func (e *Engine) completeChunkUpload(id string) (canvasResource, error) {
	v, ok := e.uploads.Load(id)
	if !ok {
		return canvasResource{}, fmt.Errorf("upload session expired")
	}
	u := v.(*canvasUpload)
	var buf []byte
	for i := 0; i < u.ChunkCount; i++ {
		part, err := os.ReadFile(filepath.Join(u.Dir, fmt.Sprintf("%06d", i)))
		if err != nil {
			return canvasResource{}, fmt.Errorf("missing chunk %d", i)
		}
		buf = append(buf, part...)
	}
	res, err := e.putCanvasBytes(u.Kind, u.Mime, u.FileName, buf, u.Width, u.Height, u.DurationMs, "")
	_ = os.RemoveAll(u.Dir)
	e.uploads.Delete(id)
	_ = e.delDoc(colUploads, id)
	return res, err
}

func (e *Engine) ServeResourceFile(id string) (string, []byte, string, error) {
	res, err := e.getCanvasResource(id)
	if err != nil {
		return "", nil, "", err
	}
	if path := e.FilePath(res.CASHash); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil, res.MimeType, nil
		}
	}
	raw, err := e.GetBytes(res.CASHash)
	return "", raw, res.MimeType, err
}
