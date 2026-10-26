// visual.go 把画幅和静帧尺寸收成各适配器实际接受的值。

package video

import "strings"

// NormalizeAspect 保留 9:16、16:9、1:1 和 adaptive。其他值收成 16:9。
func NormalizeAspect(s string) string {
	switch strings.TrimSpace(s) {
	case "9:16", "16:9", "1:1", "adaptive":
		return s
	default:
		return "16:9"
	}
}

// ImageSizeFor 给出静帧意图，不是供应商枚举。角色设定图固定横版 1920x1080，不跟集的画幅走。场景跟集的画幅：9:16 是 1080x1920，1:1 是 1024x1024，其余 1920x1080。道具是 1024x1024 的方图。
// kind 为 prop 时返回 1024x1024，为 character 时返回 1920x1080，其余把 aspect 传给 NormalizeAspect。
// aspect 在 kind 不是 prop 和 character 时传给 NormalizeAspect。
func ImageSizeFor(kind, aspect string) string {
	switch kind {
	case "prop":
		return "1024x1024"
	case "character":
		return "1920x1080"
	default:
		switch NormalizeAspect(aspect) {
		case "9:16":
			return "1080x1920"
		case "1:1":
			return "1024x1024"
		default:
			return "1920x1080"
		}
	}
}

// OpenAIImageSize 把任意宽高收成 gpt-image 接受的尺寸：正方形 1024x1024，横图 1536x1024，竖图 1024x1536。解析失败时用 1024x1024。
// size 传给 splitSize；解析失败、或宽等于高时返回 1024x1024，宽大于高返回 1536x1024，否则返回 1024x1536。
func OpenAIImageSize(size string) string {
	w, h, ok := splitSize(size)
	if !ok {
		return "1024x1024"
	}
	if w == h {
		return "1024x1024"
	}
	if w > h {
		return "1536x1024"
	}
	return "1024x1536"
}

// GeminiAspectFromSize 从宽高读出 1:1、16:9 或 9:16。解析失败时用集的画幅；fallback 是 adaptive 时用 16:9。
// size 传给 splitSize；失败时把 fallback 交给 NormalizeAspect，得到 adaptive 时返回 16:9；成功时宽等于高返回 1:1，宽大于高返回 16:9，否则返回 9:16。
func GeminiAspectFromSize(size, fallback string) string {
	w, h, ok := splitSize(size)
	if !ok {
		ar := NormalizeAspect(fallback)
		if ar == "adaptive" {
			return "16:9"
		}
		return ar
	}
	if w == h {
		return "1:1"
	}
	if w > h {
		return "16:9"
	}
	return "9:16"
}

// IsNarrator 判断这一行是不是只有声音、不生成脸部静帧。名字或角色里出现旁白、画外音、narrator、voice-over、voiceover，或列出的日文、韩文旁白词时返回真。两边都空时返回假。
// name 与 role 用空格拼起来，去掉首尾空白并转成小写，结果为空时返回假，否则看是否包含列出的旁白词。
// role 与 name 拼进同一段再查找。
func IsNarrator(name, role string) bool {
	text := strings.ToLower(strings.TrimSpace(name + " " + role))
	if text == "" {
		return false
	}
	keys := []string{"旁白", "画外音", "narrator", "voice-over", "voiceover", "ナレーター", "ナレーション", "내레이션", "해설"}
	for _, k := range keys {
		if strings.Contains(text, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// chineseContent 把空串、zh、zh-cn 和 zh-hans 当成中文。
// lang 去掉首尾空白并转成小写，空串或等于 zh、zh-cn、zh-hans 时返回真。
func chineseContent(lang string) bool {
	s := strings.ToLower(strings.TrimSpace(lang))
	return s == "" || s == "zh" || s == "zh-cn" || s == "zh-hans"
}
