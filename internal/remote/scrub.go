// 本文件按 JSON 字段名丢掉即将离开本机的密钥。不决定哪次请求要洗。
// WebSocket 的 sock.send 调用 Scrub。浏览器 HTTP 只在内容类型含 json 时调用 scrub，非 JSON 不洗。

package remote

import (
	"encoding/json"
	"strings"
)

// secretFields 是要整段丢掉的字段名。比较前先转成小写。
var secretFields = map[string]struct{}{
	"api_key": {}, "apikey": {}, "token": {}, "secret": {}, "authorization": {},
	"search_key": {}, "password": {}, "refresh_token": {}, "access_token": {},
	"private_key": {}, "api_key_value": {},
}

// Scrub 去掉值里的密钥字段。has_key 不在名单里，会留下来，手机才能显示提供商是否已配置。
// 命中的名字整段丢掉，不改写成空串，也不扫描字段值里的密钥文本。名字不区分大小写。
// 先按 JSON 往返再交给 scrub。编码失败、得到空字节，或解不开时，原样返回，密钥还在。往返之后整数会变成浮点。
func Scrub(v any) any {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 {
		return v
	}
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return v
	}
	return scrub(decoded)
}

// scrub 递归处理已经解开的 JSON。映射抄一份并跳过密钥字段。数组就地替换元素。其余类型原样返回。
func scrub(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if _, drop := secretFields[strings.ToLower(k)]; drop {
				continue
			}
			out[k] = scrub(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = scrub(t[i])
		}
		return t
	default:
		return v
	}
}
