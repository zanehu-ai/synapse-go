package config

import (
	"os"
	"strconv"
	"strings"
)

// MySQLConfig holds MySQL connection settings.
type MySQLConfig struct {
	DSN                 string
	MigrationsPath      string
	MigrateForceVersion int
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// AuthConfig holds authentication secrets.
type AuthConfig struct {
	JWTSecret   string
	AdminSecret string
}

// SMTPConfig holds SMTP mail settings.
type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// RateLimitConfig holds default rate limit values.
type RateLimitConfig struct {
	DefaultRPM         int
	DefaultTPM         int
	DefaultConcurrency int
}

// GetEnv reads an environment variable with a fallback default.
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetEnvInt reads an integer environment variable with a fallback default.
func GetEnvInt(key string, defaultVal int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetEnvBool reads a boolean environment variable with a fallback default.
func GetEnvBool(key string, defaultVal bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetEnvCSV reads a comma-separated environment variable, trimming empty items.
func GetEnvCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
