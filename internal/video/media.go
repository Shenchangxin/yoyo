package video

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func (e *Engine) PutBytes(b []byte) (string, error) {
	if e.CAS == nil {
		return "", fmt.Errorf("cas missing")
	}
	return e.CAS.PutRaw(b)
}

func (e *Engine) GetBytes(hash string) ([]byte, error) {
	if hash == "" || e.CAS == nil {
		return nil, fmt.Errorf("missing media")
	}
	return e.CAS.GetRaw(hash)
}

func (e *Engine) FilePath(hash string) string {
	if e.CAS == nil || hash == "" {
		return ""
	}
	return e.CAS.Path(hash)
}

func (e *Engine) MediaURL(hash string) string {
	if hash == "" {
		return ""
	}
	if e.MediaBase == "" {
		return ""
	}
	return strings.TrimRight(e.MediaBase, "/") + "/media/" + hash
}

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

func (e *Engine) RefDataURL(hash string) (string, error) {
	raw, err := e.GetBytes(hash)
	if err != nil {
		return "", err
	}
	jpg, err := CompressJPEG(raw, 768, 68)
	if err != nil {
		// already small jpeg maybe
		if strings.HasPrefix(http.DetectContentType(raw), "image/") {
			return "data:" + http.DetectContentType(raw) + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
		}
		return "", err
	}
	return DataURLJPEG(jpg), nil
}

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

func decodeDataURL(s string) ([]byte, error) {
	i := strings.Index(s, ",")
	if i < 0 {
		return nil, fmt.Errorf("bad data url")
	}
	return base64.StdEncoding.DecodeString(s[i+1:])
}

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
