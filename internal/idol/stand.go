package idol

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ShinyColorsDB keeps each card's Spine 3.6 stand on this CDN. eidolon-sc
// downloads the same files with httpx and draws them with spine-webgl.js;
// nothing about the picture is rendered in Python.
const standCDN = "https://cf-static.shinycolors.moe/spine/idols/"

var (
	outfitIDPattern = regexp.MustCompile(`^(stand|stand_costume)/\d+$`)
	standFiles      = map[string]bool{"data.json": true, "data.atlas": true, "data.png": true}
)

func validOutfitID(id string) bool {
	return outfitIDPattern.MatchString(id)
}

// StandCache serves one outfit's data.json, data.atlas, and data.png.
// Files are fetched once and kept under Dir.
type StandCache struct {
	Dir    string
	Base   string
	Client *http.Client

	mu      sync.Mutex
	loading map[string]*sync.Mutex
}

func (c *StandCache) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func (c *StandCache) base() string {
	if strings.TrimSpace(c.Base) != "" {
		return c.Base
	}
	return standCDN
}

func (c *StandCache) Open(outfit, name string) ([]byte, string, error) {
	if !validOutfitID(outfit) || !standFiles[name] {
		return nil, "", fmt.Errorf("bad stand path")
	}
	if strings.TrimSpace(c.Dir) == "" {
		return nil, "", fmt.Errorf("no stand cache")
	}
	key := outfit + "/" + name
	c.mu.Lock()
	if c.loading == nil {
		c.loading = map[string]*sync.Mutex{}
	}
	lock := c.loading[key]
	if lock == nil {
		lock = &sync.Mutex{}
		c.loading[key] = lock
	}
	c.mu.Unlock()
	lock.Lock()
	defer lock.Unlock()

	path, err := c.filePath(outfit, name)
	if err != nil {
		return nil, "", err
	}
	if body, err := os.ReadFile(path); err == nil && standLooksValid(name, body) {
		body = finishStand(name, body)
		return body, standContentType(name, body), nil
	}
	body, err := c.download(outfit, name)
	if err != nil {
		return nil, "", err
	}
	body = finishStand(name, body)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, "", err
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return nil, "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, "", err
	}
	return body, standContentType(name, body), nil
}

func (c *StandCache) filePath(outfit, name string) (string, error) {
	path := filepath.Join(c.Dir, filepath.FromSlash(outfit), name)
	rel, err := filepath.Rel(c.Dir, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("bad stand path")
	}
	return path, nil
}

func (c *StandCache) download(outfit, name string) ([]byte, error) {
	url := strings.TrimRight(c.base(), "/") + "/" + outfit + "/" + name
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Yoyo")
	res, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stand %s: %s", name, res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if !standLooksValid(name, body) {
		return nil, fmt.Errorf("stand %s: not a spine file", name)
	}
	return body, nil
}

func finishStand(name string, body []byte) []byte {
	if name != "data.atlas" {
		return body
	}
	return []byte(pointAtlasAtDataPNG(string(body)))
}

// pointAtlasAtDataPNG rewrites the known page slip `date.png` back to `data.png`,
// the name the texture is stored under. Other page names are left alone.
func pointAtlasAtDataPNG(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "date.png" {
			continue
		}
		next := ""
		for _, later := range lines[i+1:] {
			if strings.TrimSpace(later) != "" {
				next = strings.TrimSpace(later)
				break
			}
		}
		if strings.HasPrefix(next, "size:") {
			lines[i] = strings.Replace(line, "date.png", "data.png", 1)
		}
	}
	return strings.Join(lines, "\n")
}

func standLooksValid(name string, body []byte) bool {
	if name == "data.json" {
		return len(bytesTrimLeft(body)) > 0 && bytesTrimLeft(body)[0] == '{'
	}
	if name == "data.png" {
		return isPNG(body) || isWebP(body)
	}
	trimmed := bytesTrimLeft(body)
	return len(trimmed) > 0 && trimmed[0] != '<'
}

func standContentType(name string, body []byte) string {
	switch name {
	case "data.json":
		return "application/json; charset=utf-8"
	case "data.atlas":
		return "text/plain; charset=utf-8"
	default:
		if isWebP(body) {
			return "image/webp"
		}
		return "image/png"
	}
}

func bytesTrimLeft(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	return b[i:]
}

func isPNG(b []byte) bool {
	return len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n"
}

func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}

// StandMiddleware serves /idol-stand/<outfit>/<file> from the cache, fetching
// a missing file from the CDN. Anything else is passed on.
func StandMiddleware(dir string, next http.Handler) http.Handler {
	if next == nil {
		next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	cache := &StandCache{Dir: dir}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		outfit, name, ok := parseStandRequest(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		body, ctype, err := cache.Open(outfit, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			return
		}
		_, _ = w.Write(body)
	})
}

func parseStandRequest(path string) (outfit, name string, ok bool) {
	const prefix = "/idol-stand/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	i := strings.LastIndex(rest, "/")
	if i <= 0 || i == len(rest)-1 {
		return "", "", false
	}
	outfit, name = rest[:i], rest[i+1:]
	if !validOutfitID(outfit) || !standFiles[name] {
		return "", "", false
	}
	return outfit, name, true
}
