// Package config 负责从环境变量装载服务配置，并在启动时做必要的安全校验。
package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 是服务运行所需的全部配置。
type Config struct {
	Env      string // dev / staging / prod
	HTTPAddr string // 监听地址，如 :8080

	DBDriver string // sqlite / mysql / postgres
	DBDSN    string
	DBDebug  bool

	JWTSecret     string
	JWTIssuer     string
	UserTokenTTL  time.Duration
	GuestTokenTTL time.Duration

	// GuestEnabled 为 true 时，未携带 token 的请求会被分配一个游客身份（临时会话），
	// 用于临时画布等无需注册即可体验的场景。
	GuestEnabled bool

	// EncKey 是用户 API Key 的 AES-256-GCM 加密密钥（32 字节）。
	// 生产环境必须显式提供 NODAL_ENC_KEY；开发环境从 JWT secret 派生，保证开箱即用。
	EncKey []byte

	// LLMTimeout 是单次模型推演的整体超时（含流式输出）。
	LLMTimeout time.Duration
	// InferMaxNodes 限制单次推演最多新增多少个节点，防止模型失控刷爆画布。
	InferMaxNodes int

	// ---- 模块 5：知识库 / RAG ----
	// StorageDir 是上传文件的落盘目录。
	StorageDir string
	// UploadMaxBytes 限制单个上传文件大小。
	UploadMaxBytes int64
	// ChunkRunes / ChunkOverlap 控制文本切片粒度（按字符）。
	ChunkRunes   int
	ChunkOverlap int
	// EmbedModel 是知识库使用的向量模型 id；EmbedDim 必须与 pgvector 列维度一致。
	EmbedModel string
	EmbedDim   int
	// RAGTopK 是默认召回的片段数。
	RAGTopK int
	// MountMaxFiles 是单个画布最多挂载多少个知识库文件。
	MountMaxFiles int

	// ---- 模块 7：REPL 沙箱 ----
	// ReplEnabled 控制是否开放代码执行能力。
	ReplEnabled bool
	// ReplRuntime 可选 local（用户命名空间 + PTY）或 firecracker（外部 launcher）。
	ReplRuntime string
	// ReplTimeout 是单会话硬超时，默认且最多 30 秒。
	ReplTimeout time.Duration
	// ReplMemoryMB 通过 RLIMIT_AS 限制解释器地址空间。
	ReplMemoryMB int
	// ReplMaxSessions / ReplMaxPerUser 控制全局与单用户并发会话数。
	ReplMaxSessions int
	ReplMaxPerUser  int
	// ReplDiskMB 是会话内 tmpfs 的容量上限。
	ReplDiskMB int
	// ReplFirecrackerLauncher 是 Firecracker 运行时使用的 launcher 可执行文件。
	ReplFirecrackerLauncher string

	CORSOrigins []string
}

const defaultSecret = "dev-insecure-secret-please-change"

// Load 从环境变量读取配置。没有任何环境变量时，使用适合本地开发的默认值。
func Load() (*Config, error) {
	cfg := &Config{
		Env:      getenv("NODAL_ENV", "dev"),
		HTTPAddr: getenv("NODAL_HTTP_ADDR", ":8080"),

		DBDriver: strings.ToLower(getenv("NODAL_DB_DRIVER", "sqlite")),
		DBDSN:    getenv("NODAL_DB_DSN", "data/nodal.db"),
		DBDebug:  getbool("NODAL_DB_DEBUG", false),

		JWTSecret: getenv("NODAL_JWT_SECRET", defaultSecret),
		JWTIssuer: getenv("NODAL_JWT_ISSUER", "nodal"),

		GuestEnabled: getbool("NODAL_GUEST_ENABLED", true),
	}

	var err error
	if cfg.UserTokenTTL, err = getduration("NODAL_USER_TOKEN_TTL", 24*time.Hour); err != nil {
		return nil, fmt.Errorf("NODAL_USER_TOKEN_TTL: %w", err)
	}
	if cfg.GuestTokenTTL, err = getduration("NODAL_GUEST_TOKEN_TTL", 72*time.Hour); err != nil {
		return nil, fmt.Errorf("NODAL_GUEST_TOKEN_TTL: %w", err)
	}
	if cfg.LLMTimeout, err = getduration("NODAL_LLM_TIMEOUT", 2*time.Minute); err != nil {
		return nil, fmt.Errorf("NODAL_LLM_TIMEOUT: %w", err)
	}
	if cfg.InferMaxNodes, err = getint("NODAL_INFER_MAX_NODES", 50); err != nil {
		return nil, fmt.Errorf("NODAL_INFER_MAX_NODES: %w", err)
	}

	cfg.StorageDir = getenv("NODAL_STORAGE_DIR", "data/uploads")
	if cfg.UploadMaxBytes, err = getint64("NODAL_UPLOAD_MAX_BYTES", 10<<20); err != nil {
		return nil, fmt.Errorf("NODAL_UPLOAD_MAX_BYTES: %w", err)
	}
	if cfg.ChunkRunes, err = getint("NODAL_CHUNK_RUNES", 800); err != nil {
		return nil, fmt.Errorf("NODAL_CHUNK_RUNES: %w", err)
	}
	if cfg.ChunkOverlap, err = getint("NODAL_CHUNK_OVERLAP", 100); err != nil {
		return nil, fmt.Errorf("NODAL_CHUNK_OVERLAP: %w", err)
	}
	cfg.EmbedModel = getenv("NODAL_EMBED_MODEL", "mock/embed")
	if cfg.EmbedDim, err = getint("NODAL_EMBED_DIM", 1536); err != nil {
		return nil, fmt.Errorf("NODAL_EMBED_DIM: %w", err)
	}
	if cfg.RAGTopK, err = getint("NODAL_RAG_TOP_K", 5); err != nil {
		return nil, fmt.Errorf("NODAL_RAG_TOP_K: %w", err)
	}
	if cfg.MountMaxFiles, err = getint("NODAL_MOUNT_MAX_FILES", 5); err != nil {
		return nil, fmt.Errorf("NODAL_MOUNT_MAX_FILES: %w", err)
	}

	cfg.ReplEnabled = getbool("NODAL_REPL_ENABLED", true)
	cfg.ReplRuntime = strings.ToLower(getenv("NODAL_REPL_RUNTIME", "local"))
	if cfg.ReplTimeout, err = getduration("NODAL_REPL_TIMEOUT", 30*time.Second); err != nil {
		return nil, fmt.Errorf("NODAL_REPL_TIMEOUT: %w", err)
	}
	if cfg.ReplMemoryMB, err = getint("NODAL_REPL_MEMORY_MB", 256); err != nil {
		return nil, fmt.Errorf("NODAL_REPL_MEMORY_MB: %w", err)
	}
	if cfg.ReplMaxSessions, err = getint("NODAL_REPL_MAX_SESSIONS", 20); err != nil {
		return nil, fmt.Errorf("NODAL_REPL_MAX_SESSIONS: %w", err)
	}
	if cfg.ReplMaxPerUser, err = getint("NODAL_REPL_MAX_PER_USER", 2); err != nil {
		return nil, fmt.Errorf("NODAL_REPL_MAX_PER_USER: %w", err)
	}
	if cfg.ReplDiskMB, err = getint("NODAL_REPL_DISK_MB", 32); err != nil {
		return nil, fmt.Errorf("NODAL_REPL_DISK_MB: %w", err)
	}
	cfg.ReplFirecrackerLauncher = strings.TrimSpace(getenv("NODAL_REPL_FIRECRACKER_LAUNCHER", ""))

	origins := getenv("NODAL_CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	for _, o := range strings.Split(origins, ",") {
		if s := strings.TrimSpace(o); s != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, s)
		}
	}

	cfg.EncKey, err = loadEncKey(cfg)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProd() bool { return c.Env == "prod" || c.Env == "production" }

func (c *Config) validate() error {
	switch c.DBDriver {
	case "sqlite", "mysql", "postgres":
	default:
		return fmt.Errorf("不支持的 NODAL_DB_DRIVER: %q（可选 sqlite/mysql/postgres）", c.DBDriver)
	}
	if c.DBDriver != "sqlite" && strings.TrimSpace(c.DBDSN) == "" {
		return fmt.Errorf("%s 驱动必须提供 NODAL_DB_DSN", c.DBDriver)
	}
	if c.IsProd() {
		if c.JWTSecret == "" || c.JWTSecret == defaultSecret || len(c.JWTSecret) < 32 {
			return fmt.Errorf("生产环境必须设置长度 >= 32 的 NODAL_JWT_SECRET")
		}
		if strings.TrimSpace(os.Getenv("NODAL_ENC_KEY")) == "" {
			return fmt.Errorf("生产环境必须设置独立的 NODAL_ENC_KEY（32 字节 base64/hex）")
		}
	}
	if c.UserTokenTTL <= 0 || c.GuestTokenTTL <= 0 {
		return fmt.Errorf("token 有效期必须为正数")
	}
	if len(c.EncKey) != 32 {
		return fmt.Errorf("加密密钥长度必须为 32 字节")
	}
	if c.InferMaxNodes <= 0 || c.InferMaxNodes > 500 {
		return fmt.Errorf("NODAL_INFER_MAX_NODES 必须在 1-500 之间")
	}
	if c.ReplEnabled {
		if c.ReplRuntime != "local" && c.ReplRuntime != "firecracker" {
			return fmt.Errorf("NODAL_REPL_RUNTIME 必须是 local 或 firecracker")
		}
		if c.ReplRuntime == "firecracker" && c.ReplFirecrackerLauncher == "" {
			return fmt.Errorf("NODAL_REPL_RUNTIME=firecracker 时必须配置 NODAL_REPL_FIRECRACKER_LAUNCHER")
		}
		if c.ReplTimeout < 5*time.Second || c.ReplTimeout > 30*time.Second {
			return fmt.Errorf("NODAL_REPL_TIMEOUT 必须在 5-30 秒之间")
		}
		if c.ReplMemoryMB < 64 || c.ReplMemoryMB > 1024 {
			return fmt.Errorf("NODAL_REPL_MEMORY_MB 必须在 64-1024 之间")
		}
		if c.ReplMaxSessions < 1 || c.ReplMaxSessions > 100 {
			return fmt.Errorf("NODAL_REPL_MAX_SESSIONS 必须在 1-100 之间")
		}
		if c.ReplMaxPerUser < 1 || c.ReplMaxPerUser > 10 || c.ReplMaxPerUser > c.ReplMaxSessions {
			return fmt.Errorf("NODAL_REPL_MAX_PER_USER 必须在 1-10 之间且不超过总上限")
		}
		if c.ReplDiskMB < 8 || c.ReplDiskMB > 256 {
			return fmt.Errorf("NODAL_REPL_DISK_MB 必须在 8-256 之间")
		}
	}
	return nil
}

// loadEncKey 读取 NODAL_ENC_KEY（base64 或 64 位 hex）；未配置时从 JWT secret 派生，
// 仅用于开发环境，生产环境必须显式配置（validate 会强制要求）。
func loadEncKey(c *Config) ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("NODAL_ENC_KEY"))
	if raw == "" {
		sum := sha256.Sum256([]byte("nodal-api-key-v1:" + c.JWTSecret))
		return sum[:], nil
	}

	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, fmt.Errorf("NODAL_ENC_KEY 必须是 32 字节的 base64 或 64 位 hex 字符串")
}

func getint(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, err
	}
	return n, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getbool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return b
}

func getduration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, err
	}
	return d, nil
}

func getint64(key string, fallback int64) (int64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("必须是整数: %q", v)
	}
	return n, nil
}
