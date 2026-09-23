package video

// ProviderTemplates are non-secret starter rows. Keys go to vault, never here.
func ProviderTemplates() []Provider {
	return []Provider{
		{
			ServiceType: "image", Provider: "volcengine", Name: "Seedream",
			BaseURL: "https://ark.cn-beijing.volces.com",
			Model:   "doubao-seedream-5-0-260128", Models: `["doubao-seedream-5-0-260128","doubao-seedream-4-0-250828"]`,
			Priority: 30, IsActive: true,
		},
		{
			ServiceType: "image", Provider: "openai", Name: "OpenAI Image",
			BaseURL: "https://api.openai.com",
			Model:   "gpt-image-2", Models: `["gpt-image-2","gpt-image-1","dall-e-3"]`,
			Priority: 20, IsActive: true,
		},
		{
			ServiceType: "image", Provider: "gemini", Name: "Gemini Image",
			BaseURL: "https://generativelanguage.googleapis.com",
			Model:   "gemini-3.1-flash-image-preview", Models: `["gemini-3.1-flash-image-preview","gemini-2.5-flash-image"]`,
			Priority: 10, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "volcengine", Name: "Seedance",
			BaseURL: "https://ark.cn-beijing.volces.com",
			Model:   "doubao-seedance-2-0-mini-260615", Models: `["doubao-seedance-2-0-mini-260615"]`,
			Priority: 30, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "minimax", Name: "MiniMax Hailuo",
			BaseURL: "https://api.minimax.io",
			Model:   "MiniMax-H3", Models: `["MiniMax-H3"]`,
			Priority: 20, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "aliyun", Name: "Wan 3.0",
			BaseURL: "https://dashscope.aliyuncs.com",
			Model:   "wan3.0-video", Models: `["wan3.0-video","wan3.0-video-prime"]`,
			Priority: 10, IsActive: true,
		},
		{
			ServiceType: "tts", Provider: "openai", Name: "OpenAI Speech",
			BaseURL: "https://api.openai.com",
			Model:   "gpt-4o-mini-tts", Models: `["gpt-4o-mini-tts","tts-1-hd","tts-1"]`,
			Priority: 30, IsActive: true,
		},
		{
			ServiceType: "tts", Provider: "minimax", Name: "MiniMax Speech",
			BaseURL: "https://api.minimax.io",
			Model:   "speech-2.6-hd", Models: `["speech-2.6-hd","speech-2.6-turbo"]`,
			Priority: 20, IsActive: true,
		},
		{
			ServiceType: "tts", Provider: "volcengine", Name: "Volcengine Speech",
			BaseURL: "https://openspeech.bytedance.com",
			Model:   "seed-tts-1.1", Models: `["seed-tts-1.1"]`,
			Priority: 10, IsActive: true,
		},
	}
}

func ServiceTypes() []string {
	return []string{"image", "video", "tts"}
}
