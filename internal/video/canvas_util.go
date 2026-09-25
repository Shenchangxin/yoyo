package video

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type canvasEnv struct {
	Code    int            `json:"code"`
	Data    any            `json:"data"`
	Msg     string         `json:"msg"`
	Reason  string         `json:"reason,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

func canvasOK(data any) canvasEnv {
	if data == nil {
		data = map[string]any{}
	}
	return canvasEnv{Code: 0, Data: data}
}

func canvasFail(code int, msg, reason string) canvasEnv {
	if code == 0 {
		code = 400
	}
	if reason == "" {
		reason = "bad_request"
	}
	return canvasEnv{Code: code, Msg: msg, Reason: reason}
}

func canvasConflict(msg string) canvasEnv {
	return canvasEnv{Code: 428, Msg: msg, Reason: "failed_precondition"}
}

func parseQueryString(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		for _, pair := range strings.Split(raw, "&") {
			if pair == "" {
				continue
			}
			k, v, _ := strings.Cut(pair, "=")
			if k != "" {
				out[k] = v
			}
		}
		return out
	}
	for k, vs := range values {
		if len(vs) == 0 {
			out[k] = ""
			continue
		}
		out[k] = vs[0]
	}
	return out
}

func asMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		if m == nil {
			return map[string]any{}
		}
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func asSlice(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case nil:
		return nil
	default:
		b, _ := json.Marshal(v)
		var out []any
		_ = json.Unmarshal(b, &out)
		return out
	}
}

func strAnyMap(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func intAny(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
		return n
	}
}

func boolAny(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return false
	}
}

func decodeB64Any(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	if s == "" {
		return nil, fmt.Errorf("empty bytes")
	}
	return base64.StdEncoding.DecodeString(s)
}

func encodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func findYingcePluginDir(videoDir string) string {
	if v := strings.TrimSpace(os.Getenv("YOYO_YINGCE_PLUGINS")); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	cands := []string{
		filepath.Join(videoDir, "plugins", "yingce"),
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, "plugins", "yingce"),
			filepath.Join(dir, "..", "plugins", "yingce"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		cur := wd
		for i := 0; i < 8; i++ {
			cands = append(cands, filepath.Join(cur, "plugins", "yingce"), filepath.Join(cur, "yoyo", "plugins", "yingce"))
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}
	for _, c := range cands {
		matches, _ := filepath.Glob(filepath.Join(c, "*.yingce-plugin"))
		if len(matches) > 0 {
			return c
		}
		mans, _ := filepath.Glob(filepath.Join(c, "*", "manifest.json"))
		if len(mans) > 0 {
			return c
		}
	}
	return filepath.Join(videoDir, "plugins", "yingce")
}

func (e *Engine) hub(kind string, payload map[string]any) {
	if e.OnHub != nil {
		e.OnHub(kind, payload)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func debrandYingce(s string) string {
	if s == "" {
		return s
	}
	repls := [][2]string{
		{"影策团队", "Yoyo"},
		{"影策创作者", "Yoyo"},
		{" / 影策", ""},
		{"影策画布", "无限画布"},
		{"接入影策", "接入画布"},
		{"影策", "Yoyo"},
	}
	for _, r := range repls {
		s = strings.ReplaceAll(s, r[0], r[1])
	}
	return s
}

func publicAppearance() map[string]any {
	year := time.Now().Year()
	return map[string]any{
		"schemaVersion": 9, "brandName": "Yoyo", "brandSlug": "yoyo",
		"authHeroTitle": "Infinite canvas", "authHeroDescription": "",
		"logoUrl": "", "darkLogoUrl": "", "logoFrameEnabled": false,
		"authVideoUrl": "", "authVideoPosterUrl": "", "authVideoAutoplay": false,
		"skinId": "classic",
		"seoTitle": "Yoyo", "seoDescription": "Yoyo infinite canvas.", "seoKeywords": "",
		"footerCopyright": fmt.Sprintf("© %d Yoyo", year),
		"icpFilingEnabled": false, "icpFilingNumber": "",
		"logoConfigured": false, "darkLogoConfigured": false,
		"authVideoConfigured": false, "authVideoPosterConfigured": false,
		"configured": true, "revision": "builtin",
		"canvas": map[string]any{
			"agentName": "Yoyo", "launcherLabel": "Agent", "panelTitle": "画布助手",
			"welcomeTitle": "在这里，和{agentName}让灵感，慢慢成形",
			"welcomeDescription": "从一个想法开始，和{agentName}一起创作。",
			"inputPlaceholder": "输入操作指导；用 @ 引用画布节点，用 / 或 、 引用 Skills",
			"avatarType": "orb", "live2dResourceId": "", "live2dEntry": "", "avatarHeight": 220,
		},
	}
}

func adminAppearance() map[string]any {
	pub := publicAppearance()
	return map[string]any{
		"schemaVersion": pub["schemaVersion"], "brandName": pub["brandName"], "brandSlug": pub["brandSlug"],
		"authHeroTitle": pub["authHeroTitle"], "authHeroDescription": pub["authHeroDescription"],
		"logoResourceId": "", "darkLogoResourceId": "", "logoFrameEnabled": false,
		"authVideoResourceId": "", "authVideoPosterResourceId": "", "authVideoAutoplay": false,
		"skinId": "classic", "skinThemes": []any{},
		"seoTitle": pub["seoTitle"], "seoDescription": pub["seoDescription"], "seoKeywords": "",
		"footerCopyright": pub["footerCopyright"],
		"icpFilingEnabled": false, "icpFilingNumber": "",
		"public": pub, "configured": true, "canvas": pub["canvas"],
	}
}
