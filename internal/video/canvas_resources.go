// 行为：画布资源的登记、分片上传、访问地址和视频封面。字节在 CAS，这里只保存索引。

package video

import (
	"fmt"
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

// resourcePublicURL 拼资源的公开地址。设置了 MediaBase 时走 /resource/ 加资源 id。
// hash 在 id 为空时传给 MediaURL。
func (e *Engine) resourcePublicURL(id, hash string) string {
	if id == "" {
		return e.MediaURL(hash)
	}
	if e.MediaBase != "" {
		return strings.TrimRight(e.MediaBase, "/") + "/resource/" + id
	}
	return "/resource/" + id
}

// canvasResourceFromRow 把存储字段收成返回给前端的资源对象。
// id 写入 ID，并和 hash 一起传给 resourcePublicURL。
// hash 写入 ObjectKey 和 CASHash，并传给 resourcePublicURL。
// kind 为空时写成 file，再写入 Kind。
// mime 为空时写成 application/octet-stream，再写入 MimeType。
// bytes 写入 Size。
// width 写入 Width。
// height 写入 Height。
// duration 写入 DurationMs。
// poster 写入 PosterHash。
// playback 写入 PlaybackStatus。
// fileName 函数体没有读取。
// created 写入 CreatedAt。
// updated 写入 UpdatedAt。
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

// getCanvasResource 读取未删除的画布资源。
// id 传给 getDoc，从 colResources 取值。
func (e *Engine) getCanvasResource(id string) (canvasResource, error) {
	rec, err := getDoc[canvasResourceRec](e, colResources, id)
	if err != nil || rec.DeletedAt != "" {
		return canvasResource{}, fmt.Errorf("resource not found")
	}
	return e.canvasResourceFromRow(rec.ID, rec.CASHash, rec.Kind, rec.Mime, rec.Bytes, rec.Width, rec.Height, rec.DurationMs, rec.PosterHash, rec.PlaybackStatus, rec.FileName, rec.CreatedAt, rec.UpdatedAt), nil
}

// putCanvasBytes 把字节写入 CAS 并登记资源。视频封面改到后台补，避免堵住这次上传。
// kind 为空时按 mime 前缀写成 image、video、audio 或 file；为 image 时尝试 Thumb，为 video 且 FFMPEG 非空时异步调用 fillVideoPoster。
// mime 传给 presentableMime 后写入 Mime。
// fileName 写入 FileName。
// raw 长度为 0 时返回 empty file，否则传给 PutBytes，长度写入 Bytes。
// width 写入 Width。
// height 写入 Height。
// duration 写入 DurationMs。
// id 为空时换成 NewID，作为资源文档的键。
func (e *Engine) putCanvasBytes(kind, mime, fileName string, raw []byte, width, height, duration int, id string) (canvasResource, error) {
	if len(raw) == 0 {
		return canvasResource{}, fmt.Errorf("empty file")
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return canvasResource{}, err
	}
	mime = presentableMime(mime, raw)
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
	deferPoster := kind == "video" && strings.TrimSpace(e.FFMPEG) != ""
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
	if deferPoster {
		// 封面抽帧最多跑 8 秒 ffmpeg。画布上传和健康检查、会话刷新共用一次 RPC，所以静帧改到事后补。
		go e.fillVideoPoster(id, hash)
	}
	return e.getCanvasResource(id)
}

// fillVideoPoster 用 ffmpeg 抽一帧并写回资源的封面哈希。引擎已关闭或没有 ffmpeg 则返回。
// hash 为空时直接返回，否则传给 FilePath，再交给 Poster。
func (e *Engine) fillVideoPoster(id, hash string) {
	if e == nil || strings.TrimSpace(e.FFMPEG) == "" || id == "" || hash == "" {
		return
	}
	if e.ctx != nil && e.ctx.Err() != nil {
		return
	}
	b, err := e.Poster(e.FilePath(hash))
	if err != nil || len(b) == 0 {
		return
	}
	if e.ctx != nil && e.ctx.Err() != nil {
		return
	}
	ph, err := e.PutBytes(b)
	if err != nil || ph == "" {
		return
	}
	rec, err := getDoc[canvasResourceRec](e, colResources, id)
	if err != nil || rec.DeletedAt != "" {
		return
	}
	rec.PosterHash = ph
	rec.UpdatedAt = Now()
	_ = e.putDoc(colResources, id, rec)
}

// putCanvasResourceFromHash 给已有的 CAS 哈希登记一条画布资源。
// hash 为空时返回 cas hash missing，否则写入 CASHash。
// kind 写入 Kind。
// mime 写入 Mime。
// bytes 写入 Bytes。
// width 写入 Width。
// height 写入 Height。
// duration 写入 DurationMs。
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

// finishCanvasMedia 用任务结果哈希登记画布资源，并写成 READY。登记失败时标成可重试。
// j 的 Params 传给 parseParams，ResultHash 传给 putCanvasResourceFromHash；失败时把 ResultState 写成 FAILED_RETRYABLE，成功时写成 READY 并写回 Params。
// raw 的长度作为字节数传给 putCanvasResourceFromHash。
// video 为真时先把种类设成 video，随后仍可能被 Mode 改成 audio、video 或 image。
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

// resourceAccess 返回资源的本地访问地址，实际变体固定为 original。
// id 传给 getCanvasResource，并写入 resourceId。
// purpose 与 display 比较，随后赋给空白标识。
// variant 写入 requestedVariant；值为 playback、有封面且 purpose 为 display 时进入的分支没有语句。
func (e *Engine) resourceAccess(id, purpose, variant string) (map[string]any, error) {
	res, err := e.getCanvasResource(id)
	if err != nil {
		return nil, err
	}
	url := res.PublicURL
	if variant == "playback" && res.PosterHash != "" && purpose == "display" {
		// 仍走原始 CAS 播放，媒体服务返回原始字节。
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

// storageUsage 汇总未删除资源的字节数，总量按 1 TiB 返回。
func (e *Engine) storageUsage() map[string]any {
	var used int64
	for _, r := range loadCol[canvasResourceRec](e, colResources) {
		if r.DeletedAt == "" {
			used += r.Bytes
		}
	}
	return map[string]any{"usedBytes": used, "totalBytes": int64(1 << 40)}
}

// startChunkUpload 按 8MiB 分片创建上传会话，并记下临时目录。
// kind 写入上传记录的 Kind。
// fileName 写入 FileName。
// size 小于等于 0 时返回 invalid size，否则按 8<<20 计算分片数并写入 Size。
// width 写入 Width。
// height 写入 Height。
// duration 写入 DurationMs。
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

// putChunk 把一块分片写到上传目录，并更新已收到的块数。
// id 用来从 uploads 取出会话，并传给 getDoc 和 putDoc。
// index 小于 0 或大于等于 ChunkCount 时返回 chunk index out of range，否则按 %06d 写成文件名，并作为 got 的键。
// raw 写入该分片文件。
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

// completeChunkUpload 按序拼回分片，登记成画布资源，然后删掉临时目录。
// id 用来从 uploads 取出会话，拼完后从 uploads 删除，并传给 delDoc。
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

// ServeResourceFile 优先返回 CAS 文件路径。路径不在时再读出字节。
// id 传给 getCanvasResource。
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
