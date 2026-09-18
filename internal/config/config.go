package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment   string
	HTTPAddr      string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	JWTTTL        time.Duration
	CORSOrigin    string
	AdminName     string
	AdminEmail    string
	AdminPassword string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(value("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("REDIS_DB invalido: %w", err)
	}
	jwtTTL, err := time.ParseDuration(value("JWT_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_TTL invalido: %w", err)
	}
	cfg := Config{
		Environment:   value("APP_ENV", "development"),
		HTTPAddr:      value("HTTP_ADDR", ":8080"),
		DatabaseURL:   value("DATABASE_URL", "postgres://financial:financial@localhost:5432/financial?sslmode=disable"),
		RedisAddr:     value("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,
		JWTSecret:     value("JWT_SECRET", "desenvolvimento-local-chave-com-32-caracteres"),
		JWTTTL:        jwtTTL,
		CORSOrigin:    value("CORS_ORIGIN", "http://localhost:3000"),
		AdminName:     value("ADMIN_NAME", "Administrador"),
		AdminEmail:    value("ADMIN_EMAIL", "admin@financial.local"),
		AdminPassword: value("ADMIN_PASSWORD", "altere-esta-senha-123"),
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET deve ter ao menos 32 caracteres")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if current := os.Getenv(key); current != "" {
		return current
	}
	return fallback
}
