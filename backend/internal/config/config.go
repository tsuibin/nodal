// Package config 负责加载与校验服务端运行配置。
// 所有配置项统一从环境变量读取，本地开发额外支持从仓库根目录的 .env 加载。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 是服务端运行期配置的聚合结构。
type Config struct {
	AppEnv   string // development / production
	HTTPAddr string // 监听地址，例如 :8080
	Version  string // 构建版本，供 /healthz 返回

	DatabaseURL string
	RedisAddr   string
	RedisPass   string
	RedisDB     int

	JWTSecret     string
	JWTExpireHour int

	UploadDir   string
	MaxUploadMB int

	// AI 默认凭据：用户可以自带 Key 覆盖，这里只作为服务端兜底。
	DefaultAIProvider string
	OpenAIAPIKey      string
	OpenAIBaseURL     string
	AnthropicAPIKey   string
	DashScopeAPIKey   string

	ReplTimeoutSec int
	ReplMemLimitMB int
}

// Load 读取环境变量并返回配置。
// 找不到 .env 不视为错误，便于容器内以纯环境变量运行。
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		Version:           getEnv("APP_VERSION", "0.1.0-module0"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTExpireHour:     getEnvInt("JWT_EXPIRE_HOURS", 168),
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadMB:       getEnvInt("MAX_UPLOAD_MB", 20),
		DefaultAIProvider: getEnv("DEFAULT_AI_PROVIDER", "openai"),
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:     getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
		DashScopeAPIKey:   getEnv("DASHSCOPE_API_KEY", ""),
		ReplTimeoutSec:    getEnvInt("REPL_EXEC_TIMEOUT_SEC", 30),
		ReplMemLimitMB:    getEnvInt("REPL_MEM_LIMIT_MB", 256),
	}

	// 生产环境必须显式配置 JWT_SECRET，避免用默认弱密钥上线。
	if cfg.IsProduction() && len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("生产环境 JWT_SECRET 长度必须 ≥ 32，当前 %d", len(cfg.JWTSecret))
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev_only_insecure_secret_change_me_please_1234567890"
	}

	return cfg, nil
}

// IsProduction 判断是否生产环境。
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
