package connection

func Templates() []Connection {
	return []Connection{
		{
			Name: "OpenAI", Vendor: "openai", Protocol: "openai-chat-completions",
			Endpoint: "https://api.openai.com/v1", Capabilities: []string{CapChat},
			Models:       []string{"gpt-4.1-mini", "gpt-4.1", "o4-mini"},
			DefaultModel: map[string]string{CapChat: "gpt-4.1-mini"}, Active: true, Priority: 40,
		},
		{
			Name: "Claude", Vendor: "claude", Protocol: "openai-chat-completions",
			Endpoint: "https://api.anthropic.com/v1", Capabilities: []string{CapChat},
			Models:       []string{"claude-sonnet-4-5", "claude-opus-4-1"},
			DefaultModel: map[string]string{CapChat: "claude-sonnet-4-5"}, Active: true, Priority: 30,
		},
		{
			Name: "Gemini", Vendor: "gemini", Protocol: "openai-chat-completions",
			Endpoint: "https://generativelanguage.googleapis.com/v1beta/openai", Capabilities: []string{CapChat},
			Models:       []string{"gemini-2.5-flash", "gemini-2.5-pro"},
			DefaultModel: map[string]string{CapChat: "gemini-2.5-flash"}, Active: true, Priority: 20,
		},
		{
			Name: "DeepSeek", Vendor: "deepseek", Protocol: "openai-chat-completions",
			Endpoint: "https://api.deepseek.com", Capabilities: []string{CapChat},
			Models:       []string{"deepseek-chat", "deepseek-reasoner"},
			DefaultModel: map[string]string{CapChat: "deepseek-chat"}, Active: true,
		},
		{
			Name: "Kimi", Vendor: "kimi", Protocol: "openai-chat-completions",
			Endpoint: "https://api.moonshot.ai/v1", Capabilities: []string{CapChat},
			Models:       []string{"kimi-k2-0905"},
			DefaultModel: map[string]string{CapChat: "kimi-k2-0905"}, Active: true,
		},
		{
			Name: "Qwen", Vendor: "qwen", Protocol: "openai-chat-completions",
			Endpoint: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", Capabilities: []string{CapChat},
			Models:       []string{"qwen-plus", "qwen-max"},
			DefaultModel: map[string]string{CapChat: "qwen-plus"}, Active: true,
		},
		{
			Name: "OpenRouter", Vendor: "openrouter", Protocol: "openai-chat-completions",
			Endpoint: "https://openrouter.ai/api/v1", Capabilities: []string{CapChat},
			Models:       []string{"anthropic/claude-sonnet-4.5", "openai/gpt-4.1"},
			DefaultModel: map[string]string{CapChat: "anthropic/claude-sonnet-4.5"}, Active: true,
		},
		{
			Name: "Seedream", Vendor: "volcengine", Protocol: "volcengine-ark-seedream",
			Endpoint: "https://ark.cn-beijing.volces.com", Capabilities: []string{CapImage},
			Models:       []string{"doubao-seedream-5-0-260128", "doubao-seedream-4-0-250828"},
			DefaultModel: map[string]string{CapImage: "doubao-seedream-5-0-260128"}, Active: true, Priority: 30,
		},
		{
			Name: "OpenAI Image", Vendor: "openai", Protocol: "openai-images",
			Endpoint: "https://api.openai.com", Capabilities: []string{CapImage},
			Models:       []string{"gpt-image-2", "gpt-image-1", "dall-e-3"},
			DefaultModel: map[string]string{CapImage: "gpt-image-2"}, Active: true, Priority: 20,
		},
		{
			Name: "Gemini Image", Vendor: "gemini", Protocol: "google-gemini-image",
			Endpoint: "https://generativelanguage.googleapis.com", Capabilities: []string{CapImage},
			Models:       []string{"gemini-3.1-flash-image-preview", "gemini-2.5-flash-image"},
			DefaultModel: map[string]string{CapImage: "gemini-3.1-flash-image-preview"}, Active: true, Priority: 10,
		},
		{
			Name: "Seedance", Vendor: "volcengine", Protocol: "volcengine-ark-seedance",
			Endpoint: "https://ark.cn-beijing.volces.com", Capabilities: []string{CapVideo},
			Models:       []string{"doubao-seedance-2-0-mini-260615"},
			DefaultModel: map[string]string{CapVideo: "doubao-seedance-2-0-mini-260615"}, Active: true, Priority: 30,
		},
		{
			Name: "MiniMax Hailuo", Vendor: "minimax", Protocol: "minimax-hailuo-video-v2",
			Endpoint: "https://api.minimax.io", Capabilities: []string{CapVideo},
			Models:       []string{"MiniMax-H3"},
			DefaultModel: map[string]string{CapVideo: "MiniMax-H3"}, Active: true, Priority: 20,
		},
		{
			Name: "Wan 3.0", Vendor: "aliyun", Protocol: "dashscope-wan-video",
			Endpoint: "https://dashscope.aliyuncs.com", Capabilities: []string{CapVideo},
			Models:       []string{"wan3.0-video", "wan3.0-video-prime"},
			DefaultModel: map[string]string{CapVideo: "wan3.0-video"}, Active: true, Priority: 10,
		},
		{
			Name: "OpenAI Speech", Vendor: "openai", Protocol: "openai-audio",
			Endpoint: "https://api.openai.com", Capabilities: []string{CapSpeech},
			Models:       []string{"gpt-4o-mini-tts", "tts-1-hd", "tts-1"},
			DefaultModel: map[string]string{CapSpeech: "gpt-4o-mini-tts"}, Active: true, Priority: 30,
		},
		{
			Name: "MiniMax Speech", Vendor: "minimax", Protocol: "openai-audio",
			Endpoint: "https://api.minimax.io", Capabilities: []string{CapSpeech},
			Models:       []string{"speech-2.6-hd", "speech-2.6-turbo"},
			DefaultModel: map[string]string{CapSpeech: "speech-2.6-hd"}, Active: true, Priority: 20,
		},
		{
			Name: "Volcengine Speech", Vendor: "volcengine", Protocol: "openai-audio",
			Endpoint: "https://openspeech.bytedance.com", Capabilities: []string{CapSpeech},
			Models:       []string{"seed-tts-1.1"},
			DefaultModel: map[string]string{CapSpeech: "seed-tts-1.1"}, Active: true, Priority: 10,
		},
		{
			Name: "S3 compatible", Vendor: "s3", Protocol: "s3",
			Endpoint: "", Capabilities: []string{CapStorage},
			Active: true, Settings: map[string]any{"region": "", "bucket": "", "path_style": true, "cdn_base_url": ""},
		},
		{
			Name: "RunningHub", Vendor: "runninghub", Protocol: "runninghub",
			Endpoint: "https://www.runninghub.cn", Capabilities: []string{CapWorkflow, CapImage, CapVideo},
			Active: true,
		},
		{
			Name: "Web search", Vendor: "search", Protocol: "search",
			Endpoint: "", Capabilities: []string{CapSearch},
			Active: true,
		},
		{
			Name: "OpenTelemetry", Vendor: "otel", Protocol: "otel",
			Endpoint: "", Capabilities: []string{CapOTEL},
			Active: true,
		},
	}
}
