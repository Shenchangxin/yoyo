// 行为：画布 HTTP 信封、查询解析和字段转换，以及影策插件目录、去品牌和外观。外观 map 里的文案是返回给前端的数据。

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

// canvasOK 返回 code 为 0 的成功信封。data 为空时改成空对象。
func canvasOK(data any) canvasEnv {
	if data == nil {
		data = map[string]any{}
	}
	return canvasEnv{Code: 0, Data: data}
}

// canvasFail 返回失败信封。code 为 0 时改成 400，reason 为空时用 bad_request。
func canvasFail(code int, msg, reason string) canvasEnv {
	if code == 0 {
		code = 400
	}
	if reason == "" {
		reason = "bad_request"
	}
	return canvasEnv{Code: code, Msg: msg, Reason: reason}
}

// canvasConflict 返回 428 和 failed_precondition，表示画布版本冲突。
func canvasConflict(msg string) canvasEnv {
	return canvasEnv{Code: 428, Msg: msg, Reason: "failed_precondition"}
}

// parseQueryString 把查询串解析成单值 map。标准解析失败时按 & 和 = 切开。
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

// asMap 把值收成 map。已经是 map 则原样返回，否则经 JSON 往返。
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

// asSlice 把值收成切片。已经是切片则原样返回，否则经 JSON 往返。
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

// strAnyMap 按给定键序取出第一个非空字符串。
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

// intAny 把常见 JSON 数字或数字字符串转成 int，转不了则为 0。
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

// boolAny 把布尔值、true、1 或非零数字当成真。
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

// decodeB64Any 解码标准 Base64。带 data URL 前缀时先去掉前缀。
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

// encodeB64 把字节编成标准 Base64。
func encodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// findYingcePluginDir 查找影策插件目录，先看环境变量，再看可执行文件和当前目录附近。
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

// hub 在 OnHub 已设置时把画布事件交出去。
func (e *Engine) hub(kind string, payload map[string]any) {
	if e.OnHub != nil {
		e.OnHub(kind, payload)
	}
}

// firstNonEmpty 返回第一个去掉空白后非空的字符串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// debrandYingce 把文案里的影策品牌替换成 Yoyo 或无限画布。
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

// publicAppearance 返回内置的公开外观，品牌名是 Yoyo，面板标题是画布助手。
func publicAppearance() map[string]any {
	year := time.Now().Year()
	return map[string]any{
		"schemaVersion": 9, "brandName": "Yoyo", "brandSlug": "yoyo",
		"authHeroTitle": "Infinite canvas", "authHeroDescription": "",
		"logoUrl": "", "darkLogoUrl": "", "logoFrameEnabled": false,
		"authVideoUrl": "", "authVideoPosterUrl": "", "authVideoAutoplay": false,
		"skinId":   "classic",
		"seoTitle": "Yoyo", "seoDescription": "Yoyo infinite canvas.", "seoKeywords": "",
		"footerCopyright":  fmt.Sprintf("© %d Yoyo", year),
		"icpFilingEnabled": false, "icpFilingNumber": "",
		"logoConfigured": false, "darkLogoConfigured": false,
		"authVideoConfigured": false, "authVideoPosterConfigured": false,
		"configured": true, "revision": "builtin",
		"canvas": map[string]any{
			"agentName": "Yoyo", "launcherLabel": "Agent", "panelTitle": "画布助手",
			"welcomeTitle":       "在这里，和{agentName}让灵感，慢慢成形",
			"welcomeDescription": "从一个想法开始，和{agentName}一起创作。",
			"inputPlaceholder":   "输入操作指导；用 @ 引用画布节点，用 / 或 、 引用 Skills",
			"avatarType":         "orb", "live2dResourceId": "", "live2dEntry": "", "avatarHeight": 220,
		},
	}
}

// adminAppearance 返回管理端外观，并带上同一份公开外观。
func adminAppearance() map[string]any {
	pub := publicAppearance()
	return map[string]any{
		"schemaVersion": pub["schemaVersion"], "brandName": pub["brandName"], "brandSlug": pub["brandSlug"],
		"authHeroTitle": pub["authHeroTitle"], "authHeroDescription": pub["authHeroDescription"],
		"logoResourceId": "", "darkLogoResourceId": "", "logoFrameEnabled": false,
		"authVideoResourceId": "", "authVideoPosterResourceId": "", "authVideoAutoplay": false,
		"skinId": "classic", "skinThemes": []any{},
		"seoTitle": pub["seoTitle"], "seoDescription": pub["seoDescription"], "seoKeywords": "",
		"footerCopyright":  pub["footerCopyright"],
		"icpFilingEnabled": false, "icpFilingNumber": "",
		"public": pub, "configured": true, "canvas": pub["canvas"],
	}
}
