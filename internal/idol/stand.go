// 本文件拥有立绘缓存。一套服装只提供 data.json、data.atlas、data.png，路径必须是 stand/<数字> 或 stand_costume/<数字>。
// StandMiddleware 挂在 /idol-stand/。缺文件时向 standCDN 拉一次，写到 Dir。这是 HTTP 缓存，不调用 runtime.Run，也不调用 Brain。

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

// standCDN 是立绘文件的默认根地址。每套卡的 Spine 3.6 立绘放在这个 CDN 上。Open 的 Base 为空时用它。
const standCDN = "https://cf-static.shinycolors.moe/spine/idols/"

var (
	// outfitIDPattern 只接受 stand/<数字> 或 stand_costume/<数字>。
	outfitIDPattern = regexp.MustCompile(`^(stand|stand_costume)/\d+$`)
	// standFiles 是一套立绘允许的三个文件名。
	standFiles = map[string]bool{"data.json": true, "data.atlas": true, "data.png": true}
)

// validOutfitID 判断服装 id 是否符合 stand/<数字> 或 stand_costume/<数字>。
func validOutfitID(id string) bool {
	return outfitIDPattern.MatchString(id)
}

// StandCache 按服装提供 data.json、data.atlas 和 data.png。文件拉过一次就留在 Dir。Base 为空时用 standCDN。
type StandCache struct {
	Dir    string
	Base   string
	Client *http.Client

	mu      sync.Mutex
	loading map[string]*sync.Mutex
}

// client 返回调用方传入的 Client。没有则用 45 秒超时的默认客户端。
func (c *StandCache) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 45 * time.Second}
}

// base 返回立绘根地址。Base 空白时用 standCDN。
func (c *StandCache) base() string {
	if strings.TrimSpace(c.Base) != "" {
		return c.Base
	}
	return standCDN
}

// Open 返回一个立绘文件的字节和 Content-Type。同一路径同时只拉一次。缓存里已有且看起来像 Spine 文件时不再下载。atlas 会把已知的 date.png 页名改回 data.png。
// outfit 交给 validOutfitID。不通过时返回 bad stand path。通过后传给 filePath 和 download。
// name 须在 standFiles 里，否则返回 bad stand path。传给 filePath、standLooksValid、finishStand、standContentType 和 download。
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

// filePath 把服装和文件名拼到 Dir 下，并拒绝逃出 Dir 的路径。
// outfit 换成系统分隔符后接到 Dir 下。
// name 接在服装目录后面。相对 Dir 以 .. 开头或是绝对路径时返回错误。
func (c *StandCache) filePath(outfit, name string) (string, error) {
	path := filepath.Join(c.Dir, filepath.FromSlash(outfit), name)
	rel, err := filepath.Rel(c.Dir, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("bad stand path")
	}
	return path, nil
}

// download 用 GET 向 base 拉取一个立绘文件，最多读 32 MiB。状态不是 200 或内容不像 Spine 文件时返回错误。User-Agent 是 Yoyo。
// outfit 接在 base 后面组成 URL。
// name 接在 outfit 后面组成 URL，并传给 standLooksValid。状态不是 200 时写进错误。
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

// finishStand 对 data.atlas 调用 pointAtlasAtDataPNG，其余文件原样返回。
// name 不是 data.atlas 时原样返回 body。
// body 在 name 为 data.atlas 时转成字符串交给 pointAtlasAtDataPNG。
func finishStand(name string, body []byte) []byte {
	if name != "data.atlas" {
		return body
	}
	return []byte(pointAtlasAtDataPNG(string(body)))
}

// pointAtlasAtDataPNG 把下一行以 size: 开头的 date.png 页名改回 data.png。贴图实际存在这个名字下。其他页名不动。
// text 按换行切开。去掉空白后等于 date.png，且后面第一个非空行以 size: 开头时，把该行第一次出现的 date.png 换成 data.png。
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

// standLooksValid 粗看文件是不是 Spine 资源。json 去掉左侧空白后须以 { 开头；png 须是 PNG 或 WebP；atlas 去掉空白后不能以 < 开头。
// name 为 data.json 时看去掉左侧空白后是否以 { 开头。为 data.png 时交给 isPNG 或 isWebP。其他名字去掉左侧空白后首字节不能是 <。
// body 按 name 做上述检查。
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

// standContentType 按文件名返回 Content-Type。data.png 若是 WebP 则用 image/webp，否则 image/png。
// name 为 data.json 时返回 application/json; charset=utf-8。为 data.atlas 时返回 text/plain; charset=utf-8。
// body 在 name 不是这两种时交给 isWebP。为真则返回 image/webp，否则 image/png。
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

// bytesTrimLeft 去掉开头的空格、换行、回车和制表符。
func bytesTrimLeft(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	return b[i:]
}

// isPNG 看前 8 字节是不是 PNG 签名。
// b 长度小于 8 时返回 false。否则前 8 字节须等于 PNG 签名。
func isPNG(b []byte) bool {
	return len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n"
}

// isWebP 看是不是 RIFF 容器里的 WEBP。
func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}

// StandMiddleware 从缓存提供 /idol-stand/<服装>/<文件>。缺了就向 CDN 拉。路径对不上的请求交给 next；next 为空时用 404。只处理 GET 和 HEAD。缓存一年，并加上 nosniff。
// dir 写入 StandCache 的 Dir。
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

// parseStandRequest 从 /idol-stand/ 后面拆出服装 id 和文件名。服装或文件名不在允许名单里则 ok 为 false。
// path 须以前缀 /idol-stand/ 开头。去掉前缀后按最后一个斜杠拆成服装和文件名。
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
