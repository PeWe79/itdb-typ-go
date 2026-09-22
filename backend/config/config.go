package config

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultSessionTTLHours 登录会话（JWT）有效期的默认小时数，可通过 ITDB_SESSION_TTL_HOURS 调整
const DefaultSessionTTLHours = 12

type Config struct {
	ServerAddr   string
	DBPath       string
	UploadDir    string
	JWTSecret    string
	HistoryLimit int64
	SessionTTL   time.Duration
	CORSOrigins  []string
}

// Load 从环境变量与 .env 文件加载配置，等价于不带命令行参数覆盖的 LoadWithOverrides
func Load() Config {
	return LoadWithOverrides(nil)
}

// LoadWithOverrides 加载配置，overrides 中的非空值按语义键覆盖对应环境变量取值，
// 语义键与命令行参数一一对应：env、addr、port、db、upload_dir、jwt_secret、history_limit、session_ttl、cors_origins
func LoadWithOverrides(overrides map[string]string) Config {
	if overrides == nil {
		overrides = map[string]string{}
	}
	loadDotEnvIfPresent(overrides["env"])
	sessionTTLHours := DefaultSessionTTLHours
	if parsed, ok := parsePositiveInt(lookupValue(overrides, "session_ttl", "ITDB_SESSION_TTL_HOURS")); ok {
		sessionTTLHours = parsed
	}
	history := int64(1000)
	if parsed, ok := parsePositiveInt64(lookupValue(overrides, "history_limit", "ITDB_HISTORY_LIMIT")); ok {
		history = parsed
	}
	return Config{
		ServerAddr:   loadServerAddr(overrides),
		DBPath:       getValue(overrides, "db", "ITDB_DB_PATH", "./data/itdb.db"),
		UploadDir:    getValue(overrides, "upload_dir", "ITDB_UPLOAD_DIR", "./data/files"),
		JWTSecret:    getValue(overrides, "jwt_secret", "ITDB_JWT_SECRET", ""),
		HistoryLimit: history,
		SessionTTL:   time.Duration(sessionTTLHours) * time.Hour,
		CORSOrigins:  parseCSV(getValue(overrides, "cors_origins", "ITDB_CORS_ORIGINS", "*")),
	}
}

// lookupValue 依次从 overrides 语义键与候选环境变量中取第一个非空值
func lookupValue(overrides map[string]string, overrideKey string, envKeys ...string) string {
	if value := strings.TrimSpace(overrides[overrideKey]); value != "" {
		return value
	}
	for _, key := range envKeys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

// getValue 在 lookupValue 基础上提供默认值回退
func getValue(overrides map[string]string, overrideKey, envKey, fallback string) string {
	if value := lookupValue(overrides, overrideKey, envKey); value != "" {
		return value
	}
	return fallback
}

// parsePositiveInt 解析正整数字符串，无效或非正值返回 false
func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// parsePositiveInt64 解析正整数 int64 字符串，无效或非正值返回 false
func parsePositiveInt64(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// loadServerAddr 按 -addr、ITDB_SERVER_ADDR、ADDR、PORT 的顺序解析监听地址
func loadServerAddr(overrides map[string]string) string {
	if addr := lookupValue(overrides, "addr", "ITDB_SERVER_ADDR"); addr != "" {
		return addr
	}
	if addr := lookupValue(overrides, "", "ADDR"); addr != "" {
		return addr
	}
	port := lookupValue(overrides, "port", "PORT")
	if port == "" {
		port = "8080"
	}
	return net.JoinHostPort("127.0.0.1", port)
}

// loadDotEnvIfPresent 按自定义路径、./.env、backend/.env、可执行文件同目录顺序加载首个存在的 .env 文件
func loadDotEnvIfPresent(customPath string) {
	seen := map[string]struct{}{}
	candidates := []string{}
	if customPath != "" {
		candidates = append(candidates, customPath)
	}
	candidates = append(candidates, ".env", filepath.Join("backend", ".env"))
	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), ".env"))
	}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if loadDotEnvFile(candidate) == nil {
			return
		}
	}
}

// loadDotEnvFile 逐行解析 .env 文件并写入环境变量，已存在的环境变量不被覆盖
func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 && ((strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"))) {
			value = value[1 : len(value)-1]
		}
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// parseCSV 解析逗号分隔的来源列表，空列表回退为通配 *
func parseCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
