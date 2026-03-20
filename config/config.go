package config

import "os"

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
