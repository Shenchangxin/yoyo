package video

// ProviderTemplates are non-secret starter rows. Keys go to vault, never here.
func ProviderTemplates() []Provider {
	return []Provider{
		{
			ServiceType: "image", Provider: "volcengine", Name: "Seedream",
			BaseURL: "https://ark.cn-beijing.volces.com", VaultKey: "video.image.volcengine",
			Model: "doubao-seedream-4-0-250828", Models: `["doubao-seedream-4-0-250828"]`,
			Priority: 30, IsActive: true,
		},
		{
			ServiceType: "image", Provider: "openai", Name: "OpenAI Image",
			BaseURL: "https://api.openai.com", VaultKey: "video.image.openai",
			Model: "gpt-image-1", Models: `["gpt-image-1","dall-e-3"]`,
			Priority: 20, IsActive: true,
		},
		{
			ServiceType: "image", Provider: "gemini", Name: "Gemini Image",
			BaseURL: "https://generativelanguage.googleapis.com", VaultKey: "video.image.gemini",
			Model: "gemini-3.1-flash-image-preview", Models: `["gemini-3.1-flash-image-preview","gemini-2.5-flash-image"]`,
			Priority: 10, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "volcengine", Name: "Seedance",
			BaseURL: "https://ark.cn-beijing.volces.com", VaultKey: "video.video.volcengine",
			Model: "doubao-seedance-2-0-mini-260615", Models: `["doubao-seedance-2-0-mini-260615"]`,
			Priority: 30, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "minimax", Name: "MiniMax Hailuo",
			BaseURL: "https://api.minimax.io", VaultKey: "video.video.minimax",
			Model: "MiniMax-H3", Models: `["MiniMax-H3"]`,
			Priority: 20, IsActive: true,
		},
		{
			ServiceType: "video", Provider: "aliyun", Name: "Wan 3.0",
			BaseURL: "https://dashscope.aliyuncs.com", VaultKey: "video.video.aliyun",
			Model: "wan3.0-video", Models: `["wan3.0-video","wan3.0-video-prime"]`,
			Priority: 10, IsActive: true,
		},
	}
}
