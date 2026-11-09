// icon.go 拥有 agent 图标：下载注册表里的 SVG，用 encoding/xml 清洗（只留基本形状和几何、填充、描边属性），
// 不超过 8 KB，缓存到 buddies/icons/<id>.svg，以 data URI 交给界面当遮罩用。脚本、链接、外部引用一律丢掉。

package buddy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxIconBytes 是清洗后图标的上限。
const maxIconBytes = 8 << 10

// iconElements 是保留的元素。
var iconElements = map[string]bool{"svg": true, "g": true, "path": true, "circle": true, "rect": true, "polygon": true, "line": true, "ellipse": true}

// iconAttrs 是保留的属性：几何、填充和描边。
var iconAttrs = map[string]bool{
	"viewBox": true, "width": true, "height": true, "xmlns": true, "d": true, "x": true, "y": true, "x1": true, "y1": true, "x2": true, "y2": true,
	"cx": true, "cy": true, "r": true, "rx": true, "ry": true, "points": true, "transform": true,
	"fill": true, "fill-rule": true, "fill-opacity": true, "clip-rule": true, "opacity": true,
	"stroke": true, "stroke-width": true, "stroke-linecap": true, "stroke-linejoin": true, "stroke-opacity": true, "stroke-miterlimit": true,
}

// errIcon 是图标不合格。
var errIcon = errors.New("buddy: unusable icon")

// sanitizeSVG 重写一份只含白名单元素和属性的 SVG。根元素必须是 svg，值里不许有 url(、javascript: 或 data:。
func sanitizeSVG(raw []byte) ([]byte, error) {
	if len(raw) > 64<<10 {
		return nil, errIcon
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = false
	var out bytes.Buffer
	depth, skip := 0, 0
	sawRoot := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errIcon
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if skip > 0 {
				skip++
				continue
			}
			name := t.Name.Local
			if !sawRoot {
				if name != "svg" {
					return nil, errIcon
				}
				sawRoot = true
			}
			if !iconElements[name] {
				skip = 1
				continue
			}
			out.WriteString("<" + name)
			if name == "svg" {
				out.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
			}
			for _, a := range t.Attr {
				key := a.Name.Local
				if a.Name.Space != "" || key == "xmlns" || !iconAttrs[key] || unsafeValue(a.Value) {
					continue
				}
				out.WriteString(" " + key + `="`)
				_ = xml.EscapeText(&out, []byte(a.Value))
				out.WriteString(`"`)
			}
			out.WriteString(">")
		case xml.EndElement:
			depth--
			if skip > 0 {
				skip--
				continue
			}
			out.WriteString("</" + t.Name.Local + ">")
		}
	}
	if !sawRoot || depth != 0 || out.Len() > maxIconBytes {
		return nil, errIcon
	}
	return out.Bytes(), nil
}

// unsafeValue 判断属性值里有没有外部引用或脚本。
func unsafeValue(v string) bool {
	lower := strings.ToLower(strings.ReplaceAll(v, " ", ""))
	return strings.Contains(lower, "url(") || strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") || strings.Contains(lower, "expression(")
}

// iconURI 把清洗过的 SVG 变成 data URI。
func iconURI(svg []byte) string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(svg)
}

// iconPath 是缓存文件的路径。
func (m *Manager) iconPath(id string) string {
	return filepath.Join(m.opts.Root, "icons", safeName(id)+".svg")
}

// loadIconsLocked 读缓存的图标。调用方持有 m.mu。
func (m *Manager) loadIconsLocked() {
	for _, e := range m.entries {
		if raw, err := os.ReadFile(m.iconPath(e.ID)); err == nil {
			if clean, err := sanitizeSVG(raw); err == nil {
				m.icons[e.ID] = iconURI(clean)
			}
		}
	}
}

// fetchIcons 下载还没缓存的图标，最多同时 4 个。只认 https。
func (m *Manager) fetchIcons(ctx context.Context) {
	if m.opts.Root == "" {
		return
	}
	m.mu.Lock()
	type want struct{ id, url string }
	var list []want
	for _, e := range m.entries {
		if e.Icon != "" && m.icons[e.ID] == "" {
			list = append(list, want{e.ID, e.Icon})
		}
	}
	m.mu.Unlock()
	sem := make(chan struct{}, 4)
	done := make(chan struct{}, len(list))
	for _, w := range list {
		w := w
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; done <- struct{}{} }()
			svg, err := m.downloadIcon(ctx, w.url)
			if err != nil {
				return
			}
			_ = os.MkdirAll(filepath.Dir(m.iconPath(w.id)), 0o755)
			_ = os.WriteFile(m.iconPath(w.id), svg, 0o644)
			m.mu.Lock()
			m.icons[w.id] = iconURI(svg)
			m.mu.Unlock()
			m.publish(w.id)
		}()
	}
	for range list {
		<-done
	}
}

// downloadIcon 下载并清洗一个图标，10 秒超时，原文不超过 64 KB。
func (m *Manager) downloadIcon(ctx context.Context, raw string) ([]byte, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return nil, errIcon
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.opts.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errIcon
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10+1))
	if err != nil {
		return nil, err
	}
	return sanitizeSVG(body)
}
