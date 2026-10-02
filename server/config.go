package server

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 应用配置，全部通过环境变量注入
type Config struct {
	Env          string
	Port         string
	ScenarioFile string
	MySQL        MySQLConfig
	JWT          JWTConfig
	WeChat       WeChatConfig
	AI           AIConfig
}

// AIConfig 大模型适配层配置
//
// Provider 为 mock 或 deepseek。未配置密钥时自动降级为 mock，
// 便于本地无外网、无预算时完整跑通对话流程。
type AIConfig struct {
	Provider    string
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	Timeout     time.Duration
	MaxRetries  int
}

// MySQLConfig MySQL 连接配置
type MySQLConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	Charset         string
	LogLevel        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN 生成 gorm mysql 驱动所需的连接串
func (m MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database, m.Charset,
	)
}

// JWTConfig 令牌签发配置
type JWTConfig struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

// WeChatConfig 微信小程序配置
//
// Mock 为 true 时不请求微信服务器，仅用于本地开发联调。
type WeChatConfig struct {
	AppID     string
	AppSecret string
	Mock      bool
}

// LoadConfig 从环境变量读取配置并填充默认值。
// 若存在 .env 文件则先加载（不覆盖已存在的环境变量）。
func LoadConfig() *Config {
	loadDotEnv(".env")
	return &Config{
		Env:          getEnv("APP_ENV", "dev"),
		Port:         getEnv("APP_PORT", "8080"),
		ScenarioFile: getEnv("SCENARIO_FILE", "config/scenarios.json"),
		MySQL: MySQLConfig{
			Host:            getEnv("MYSQL_HOST", "127.0.0.1"),
			Port:            getEnv("MYSQL_PORT", "3306"),
			User:            getEnv("MYSQL_USER", "root"),
			Password:        getEnv("MYSQL_PASSWORD", ""),
			Database:        getEnv("MYSQL_DATABASE", "starlang_bridge"),
			Charset:         getEnv("MYSQL_CHARSET", "utf8mb4"),
			LogLevel:        getEnv("MYSQL_LOG_LEVEL", "warn"),
			MaxOpenConns:    getEnvInt("MYSQL_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    getEnvInt("MYSQL_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvInt("MYSQL_CONN_MAX_LIFETIME_MIN", 60)) * time.Minute,
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "starlang-bridge-dev-secret"),
			Issuer: getEnv("JWT_ISSUER", "starlang-bridge"),
			TTL:    time.Duration(getEnvInt("JWT_TTL_HOURS", 72)) * time.Hour,
		},
		WeChat: WeChatConfig{
			AppID:     getEnv("WECHAT_APP_ID", ""),
			AppSecret: getEnv("WECHAT_APP_SECRET", ""),
			Mock:      getEnvBool("WECHAT_MOCK", false),
		},
		AI: AIConfig{
			Provider:    getEnv("AI_PROVIDER", "mock"),
			APIKey:      getEnv("DEEPSEEK_API_KEY", ""),
			BaseURL:     getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			Model:       getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
			Temperature: getEnvFloat("AI_TEMPERATURE", 0.3),
			Timeout:     time.Duration(getEnvInt("AI_TIMEOUT_SECONDS", 15)) * time.Second,
			MaxRetries:  getEnvInt("AI_MAX_RETRIES", 2),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// loadDotEnv 读取简单的 KEY=VALUE 文件，仅在环境变量尚未设置时生效
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
