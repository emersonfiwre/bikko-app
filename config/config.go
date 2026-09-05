package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port string
	Env  string

	// Postgres DB Config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSchema   string

	// Redis Config
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Cloudflare R2 Config
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2PublicURL  string
}

func LoadConfig() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "bikko"),
		DBPassword: getEnv("DB_PASSWORD", "bikkopass"),
		DBName:     getEnv("DB_NAME", "bikkodb"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		R2AccountID:  getEnv("R2_ACCOUNT_ID", "mock_r2_account"),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", "mock_r2_access_key"),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", "mock_r2_secret_key"),
		R2BucketName: getEnv("R2_BUCKET_NAME", "bikko-media"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", "https://pub-bikko.r2.dev"),
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
