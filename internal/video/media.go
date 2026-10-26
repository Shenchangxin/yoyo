// media.go 把字节和本地文件放进内容寻址存储，并拼出媒体地址。

package video

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PutBytes 把字节写入 CAS。CAS 为空时返回 cas missing。
func (e *Engine) PutBytes(b []byte) (string, error) {
	if e.CAS == nil {
		return "", fmt.Errorf("cas missing")
	}
	return e.CAS.PutRaw(b)
}

// maxMediaFile 是单个用户文件的上限，4 GiB。
const maxMediaFile = 4 << 30

// PutPath 把用户选中的文件抄进 CAS，不把整文件读进网页视图。空路径、非普通文件、空文件，以及大于 4 GiB 的文件都会拒绝。能用 PutFile 时按路径写入，否则整文件读入再 PutBytes。返回哈希和按扩展名得到的 MIME，未知扩展名用 application/octet-stream。
func (e *Engine) PutPath(path string) (string, string, error) {
	if e.CAS == nil {
		return "", "", fmt.Errorf("cas missing")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", fmt.Errorf("path required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", "", err
	}
	if !st.Mode().IsRegular() {
		return "", "", fmt.Errorf("not a file")
	}
	if st.Size() <= 0 {
		return "", "", fmt.Errorf("file is empty")
	}
	if st.Size() > maxMediaFile {
		return "", "", fmt.Errorf("file is larger than 4GB")
	}
	kind := mime.TypeByExtension(strings.ToLower(filepath.Ext(abs)))
	if kind == "" {
		kind = "application/octet-stream"
	}
	if store, ok := e.CAS.(interface{ PutFile(string) (string, error) }); ok {
		hash, err := store.PutFile(abs)
		if err != nil {
			return "", "", err
		}
		return hash, kind, nil
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", "", err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return "", "", err
	}
	return hash, kind, nil
}

// GetBytes 按哈希读回字节。哈希不安全或 CAS 为空时返回 missing media。
func (e *Engine) GetBytes(hash string) ([]byte, error) {
	if !safeCASHash(hash) || e.CAS == nil {
		return nil, fmt.Errorf("missing media")
	}
	return e.CAS.GetRaw(strings.TrimSpace(hash))
}

// safeCASHash 拒绝空串、含 .. 或含斜杠、反斜杠的哈希。
func safeCASHash(hash string) bool {
	h := strings.TrimSpace(hash)
	if h == "" || strings.Contains(h, "..") || strings.ContainsAny(h, `/\`) {
		return false
	}
	return true
}

// FilePath 返回哈希对应的本地路径。CAS 为空或哈希不安全时返回空串。
func (e *Engine) FilePath(hash string) string {
	if e.CAS == nil || !safeCASHash(hash) {
		return ""
	}
	return e.CAS.Path(strings.TrimSpace(hash))
}

// casFile 返回磁盘上已存在、非空的普通文件路径。调用方探测这条路径，而不是把整段视频读进内存。
func (e *Engine) casFile(hash string) string {
	path := e.FilePath(hash)
	if path == "" {
		return ""
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() <= 0 {
		return ""
	}
	return path
}

// MediaURL 在 MediaBase 和哈希都非空时拼出 /media/ 地址。
func (e *Engine) MediaURL(hash string) string {
	if hash == "" {
		return ""
	}
	if e.MediaBase == "" {
		return ""
	}
	return strings.TrimRight(e.MediaBase, "/") + "/media/" + hash
}

// Thumb 把图片压成宽高不超过 400、质量 70 的 JPEG，再写回 CAS。
func (e *Engine) Thumb(hash string) (string, error) {
	raw, err := e.GetBytes(hash)
	if err != nil {
		return "", err
	}
	jpg, err := CompressJPEG(raw, 400, 70)
	if err != nil {
		return "", err
	}
	return e.PutBytes(jpg)
}

// RefDataURL 把图片压成宽高不超过 768、质量 68 的 JPEG 数据 URL。压不成且原字节是图片时，按探测到的类型直接做成数据 URL。
func (e *Engine) RefDataURL(hash string) (string, error) {
	raw, err := e.GetBytes(hash)
	if err != nil {
		return "", err
	}
	jpg, err := CompressJPEG(raw, 768, 68)
	if err != nil {
		if strings.HasPrefix(http.DetectContentType(raw), "image/") {
			return "data:" + http.DetectContentType(raw) + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
		}
		return "", err
	}
	return DataURLJPEG(jpg), nil
}

// DownloadURL 读取 data: URL，或用 Engine.HTTP 做 GET。状态码大于等于 400 时失败。正文最多读 80 MiB。
func (e *Engine) DownloadURL(rawURL string) ([]byte, error) {
	if strings.HasPrefix(rawURL, "data:") {
		return decodeDataURL(rawURL)
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := e.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("download %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 80<<20))
}

// StoreUpload 写入字节，缩略图能生成时一并返回。缩略图失败不影响哈希。
func (e *Engine) StoreUpload(b []byte) (hash, thumb string, err error) {
	hash, err = e.PutBytes(b)
	if err != nil {
		return "", "", err
	}
	if t, e2 := e.Thumb(hash); e2 == nil {
		thumb = t
	}
	return hash, thumb, nil
}

// decodeDataURL 取逗号后面的标准 base64。没有逗号时返回 bad data url。
func decodeDataURL(s string) ([]byte, error) {
	i := strings.Index(s, ",")
	if i < 0 {
		return nil, fmt.Errorf("bad data url")
	}
	return base64.StdEncoding.DecodeString(s[i+1:])
}

// ServeFile 返回哈希对应且确实存在的本地路径。路径为空时返回 missing。
func (e *Engine) ServeFile(hash string) (path string, err error) {
	p := e.FilePath(hash)
	if p == "" {
		return "", fmt.Errorf("missing")
	}
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}
