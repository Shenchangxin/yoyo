// 连接文档的字段和能力名。VaultKey 只保存钥匙名字。

package connection

import (
	"encoding/json"
	"strings"
)

const (
	// CapChat 是聊天能力，值为 chat。
	CapChat = "chat"
	// CapImage 是图像能力，值为 image。
	CapImage = "image"
	// CapVideo 是视频能力，值为 video。
	CapVideo = "video"
	// CapSpeech 是语音合成能力，值为 speech。
	CapSpeech = "speech"
	// CapMusic 是音乐能力，值为 music。
	CapMusic = "music"
	// CapMatting 是抠图能力，值为 matting。
	CapMatting = "matting"
	// CapStock 是素材库能力，值为 stock。
	CapStock = "stock"
	// CapASR 是语音识别能力，值为 asr。
	CapASR = "asr"
	// CapSearch 是搜索能力，值为 search。
	CapSearch = "search"
	// CapStorage 是对象存储能力，值为 storage。
	CapStorage = "storage"
	// CapWorkflow 是工作流能力，值为 workflow。
	CapWorkflow = "workflow"
	// CapOTEL 是 OpenTelemetry，值为 otel。
	CapOTEL = "otel"
	// CapMCP 是 MCP，值为 mcp。
	CapMCP = "mcp"
	// CapProtocol 是协议标记，值为 protocol。CapsOf 会跳过它。
	CapProtocol = "protocol"
)

// Connection 是一条供应商连接。VaultKey 是钥匙名字。没有存放密钥原文的字段。HasKey 由读取时向仓库查询，写入 JSON 前被清成 false。
type Connection struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Vendor       string            `json:"vendor"`
	Protocol     string            `json:"protocol"`
	Endpoint     string            `json:"endpoint"`
	VaultKey     string            `json:"vault_key"`
	Capabilities []string          `json:"capabilities"`
	Models       []string          `json:"models"`
	DefaultModel map[string]string `json:"default_model"`
	Active       bool              `json:"active"`
	Priority     int               `json:"priority"`
	Settings     map[string]any    `json:"settings,omitempty"`
	HasKey       bool              `json:"has_key"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// Defaults 把能力名映射到默认连接 id。
type Defaults map[string]string

// NormalizeCap 把别名收成能力常量。不认识的字符串转成小写后原样返回。
func NormalizeCap(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "llm", "text", "chat":
		return CapChat
	case "tts", "audio", "speech":
		return CapSpeech
	case "bgm", "soundtrack", "score", "music":
		return CapMusic
	case "matte", "matting", "cutout", "抠像":
		return CapMatting
	case "stock", "pexels", "pixabay", "unsplash", "giphy", "footage":
		return CapStock
	case "asr", "whisper", "stt", "transcribe", "transcription":
		return CapASR
	case "runninghub", "workflow":
		return CapWorkflow
	case "oss", "s3", "cas", "storage":
		return CapStorage
	default:
		return s
	}
}

// HasCap 报告连接声明的能力里有没有 cap。比较前两边都 NormalizeCap。
func HasCap(c Connection, cap string) bool {
	want := NormalizeCap(cap)
	for _, x := range c.Capabilities {
		if NormalizeCap(x) == want {
			return true
		}
	}
	return false
}

// ModelFor 先看 DefaultModel 里的该能力，再看空键，最后用 Models 的第一项。都没有则返回空串。
func ModelFor(c Connection, cap string) string {
	cap = NormalizeCap(cap)
	if c.DefaultModel != nil {
		if v := strings.TrimSpace(c.DefaultModel[cap]); v != "" {
			return v
		}
		if v := strings.TrimSpace(c.DefaultModel[""]); v != "" {
			return v
		}
	}
	if len(c.Models) > 0 {
		return c.Models[0]
	}
	return ""
}

// ModelsJSON 把模型列表编成 JSON 数组。列表为空时，若 ModelFor 有值就编成单元素数组，否则是 []。
func ModelsJSON(c Connection) string {
	if len(c.Models) == 0 {
		if m := ModelFor(c, ""); m != "" {
			b, _ := json.Marshal([]string{m})
			return string(b)
		}
		return "[]"
	}
	b, _ := json.Marshal(c.Models)
	return string(b)
}

// ParseModels 解析 JSON 数组或逗号分隔的模型名。空白返回 nil。
func ParseModels(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if json.Unmarshal([]byte(raw), &arr) == nil {
			return compact(arr)
		}
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		out = append(out, strings.TrimSpace(p))
	}
	return compact(out)
}

// compact 去掉空白和重复项，保留第一次出现的顺序。
func compact(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// InferProtocol 按能力和供应商给出协议字符串。未列出的聊天供应商用 openai-chat-completions。
func InferProtocol(vendor, cap string) string {
	v := strings.ToLower(strings.TrimSpace(vendor))
	c := NormalizeCap(cap)
	switch c {
	case CapStorage:
		if v == "cas" || v == "" {
			return "cas"
		}
		return "s3"
	case CapSearch:
		switch v {
		case "tavily":
			return "tavily"
		case "brave":
			return "brave"
		case "serper":
			return "serper"
		case "bocha":
			return "bocha"
		case "jina":
			return "jina"
		case "exa":
			return "exa"
		case "duckduckgo", "ddg":
			return "duckduckgo"
		case "metaso":
			return "search"
		default:
			return "search"
		}
	case CapOTEL:
		return "otel"
	case CapMCP:
		return "mcp-http"
	case CapWorkflow:
		return "runninghub"
	case CapImage:
		switch v {
		case "openai":
			return "openai-images"
		case "gemini":
			return "google-gemini-image"
		case "volcengine":
			return "volcengine-ark-seedream"
		case "jimeng":
			return "volcengine-jimeng-image"
		case "tencent", "hunyuan":
			return "hunyuan-image"
		default:
			return "openai-images"
		}
	case CapVideo:
		switch v {
		case "openai":
			return "openai-videos"
		case "gemini":
			return "google-gemini-veo"
		case "minimax", "hailuo":
			return "minimax-hailuo-video-v2"
		case "aliyun", "wan", "dashscope":
			return "dashscope-wan-video"
		case "volcengine", "seedance":
			return "volcengine-ark-seedance"
		case "kling", "kuaishou", "kolors":
			return "openai-videos"
		case "runway":
			return "openai-videos"
		case "jimeng":
			return "volcengine-jimeng-video"
		case "vidu":
			return "vidu"
		case "tencent", "hunyuan":
			return "hunyuan-video"
		default:
			return "openai-videos"
		}
	case CapSpeech:
		switch v {
		case "azure", "microsoft":
			return "openai-audio"
		case "elevenlabs":
			return "openai-audio"
		default:
			return "openai-audio"
		}
	case CapMatting:
		switch v {
		case "removebg", "remove.bg":
			return "removebg"
		case "photoroom":
			return "photoroom"
		case "volcengine", "jimeng":
			return "volcengine-visual-matting"
		default:
			return "volcengine-visual-matting"
		}
	case CapMusic:
		switch v {
		case "minimax", "hailuo":
			return "minimax-music"
		case "elevenlabs":
			return "elevenlabs-sound"
		case "stability":
			return "stability-audio"
		case "suno":
			return "suno"
		case "gemini", "google":
			return "google-lyria"
		case "mureka", "skywork":
			return "mureka"
		default:
			return "openai-audio"
		}
	case CapStock:
		switch v {
		case "pixabay":
			return "pixabay"
		case "unsplash":
			return "unsplash"
		case "giphy":
			return "giphy"
		default:
			return "pexels"
		}
	case CapASR:
		switch v {
		case "deepgram":
			return "deepgram"
		case "assemblyai":
			return "assemblyai"
		case "azure", "microsoft":
			return "azure-stt"
		case "aliyun", "dashscope", "qwen":
			return "dashscope-asr"
		case "tencent", "hunyuan":
			return "tencent-asr"
		default:
			return "openai-audio"
		}
	default:
		switch v {
		case "claude", "anthropic":
			return "openai-chat-completions"
		case "gemini":
			return "openai-chat-completions"
		default:
			return "openai-chat-completions"
		}
	}
}
