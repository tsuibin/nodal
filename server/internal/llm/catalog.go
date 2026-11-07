package llm

// builtinCatalog 是内置模型目录。
//
// 所有 OpenAI 兼容的厂商（openai / deepseek / moonshot / zhipu / qwen）共用同一个
// openaiClient，差异只在默认 BaseURL 与 upstream model 名；用户可以在凭证里覆盖
// BaseURL 接入自建/代理网关。需要新增厂商时，往这里加一条 Spec + 注册 client 即可。
func builtinCatalog() []Spec {
	return []Spec{
		{
			ID:            "mock/basic",
			Label:         "本地模拟（开发/演示）",
			Provider:      "mock",
			UpstreamModel: "mock",
			MaxTokens:     4096,
			RequiresKey:   false,
		},
		{
			ID:             "openai/gpt-4o-mini",
			Label:          "OpenAI GPT-4o mini",
			Provider:       "openai",
			UpstreamModel:  "gpt-4o-mini",
			DefaultBaseURL: "https://api.openai.com/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "openai/gpt-4o",
			Label:          "OpenAI GPT-4o",
			Provider:       "openai",
			UpstreamModel:  "gpt-4o",
			DefaultBaseURL: "https://api.openai.com/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "deepseek/deepseek-chat",
			Label:          "DeepSeek Chat",
			Provider:       "deepseek",
			UpstreamModel:  "deepseek-chat",
			DefaultBaseURL: "https://api.deepseek.com/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "deepseek/deepseek-reasoner",
			Label:          "DeepSeek Reasoner",
			Provider:       "deepseek",
			UpstreamModel:  "deepseek-reasoner",
			DefaultBaseURL: "https://api.deepseek.com/v1",
			MaxTokens:      8192,
			RequiresKey:    true,
		},
		{
			ID:             "moonshot/kimi-k2",
			Label:          "Moonshot Kimi K2",
			Provider:       "moonshot",
			UpstreamModel:  "kimi-k2",
			DefaultBaseURL: "https://api.moonshot.cn/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "zhipu/glm-4.5",
			Label:          "智谱 GLM-4.5",
			Provider:       "zhipu",
			UpstreamModel:  "glm-4.5",
			DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "qwen/qwen-plus",
			Label:          "通义千问 Plus",
			Provider:       "qwen",
			UpstreamModel:  "qwen-plus",
			DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
		{
			ID:             "anthropic/claude-3-5-sonnet",
			Label:          "Anthropic Claude 3.5 Sonnet",
			Provider:       "anthropic",
			UpstreamModel:  "claude-3-5-sonnet-20241022",
			DefaultBaseURL: "https://api.anthropic.com/v1",
			MaxTokens:      4096,
			RequiresKey:    true,
		},
	}
}
