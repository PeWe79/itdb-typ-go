package config

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ServerAddr   string
	DBPath       string
	UploadDir    string
	JWTSecret    string
	HistoryLimit int64
	CORSOrigins  []string
}

func Load() Config {
	loadDotEnvIfPresent()
	history := int64(1000)
	if value := strings.TrimSpace(os.Getenv("ITDB_HISTORY_LIMIT")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			history = parsed
		}
	}
	return Config{
		ServerAddr:   loadServerAddr(),
		DBPath:       getenv("ITDB_DB_PATH", "./data/itdb.db"),
		UploadDir:    getenv("ITDB_UPLOAD_DIR", "./data/files"),
		JWTSecret:    getenv("ITDB_JWT_SECRET", ""),
		HistoryLimit: history,
		CORSOrigins:  parseCSV(getenv("ITDB_CORS_ORIGINS", "*")),
	}
}

func loadServerAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ITDB_SERVER_ADDR")); addr != "" {
		return addr
	}
	if addr := strings.TrimSpace(os.Getenv("ADDR")); addr != "" {
		return addr
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	return net.JoinHostPort("127.0.0.1", port)
}

func loadDotEnvIfPresent() {
	seen := map[string]struct{}{}
	candidates := []string{".env", filepath.Join("backend", ".env")}
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

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

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
