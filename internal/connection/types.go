package connection

import (
	"encoding/json"
	"strings"
)

const (
	CapChat     = "chat"
	CapImage    = "image"
	CapVideo    = "video"
	CapSpeech   = "speech"
	CapSearch   = "search"
	CapStorage  = "storage"
	CapWorkflow = "workflow"
	CapOTEL     = "otel"
	CapMCP      = "mcp"
	CapProtocol = "protocol"
)

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

type Defaults map[string]string

func NormalizeCap(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "llm", "text", "chat":
		return CapChat
	case "tts", "audio", "speech":
		return CapSpeech
	case "runninghub", "workflow":
		return CapWorkflow
	case "oss", "s3", "cas", "storage":
		return CapStorage
	default:
		return s
	}
}

func HasCap(c Connection, cap string) bool {
	want := NormalizeCap(cap)
	for _, x := range c.Capabilities {
		if NormalizeCap(x) == want {
			return true
		}
	}
	return false
}

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
		return "search"
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
		default:
			return "openai-videos"
		}
	case CapSpeech:
		return "openai-audio"
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
