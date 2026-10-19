package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string
	Env  string

	JWTSecret string

	// Postgres DB Config
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSchema            string
	DBSSLMode           string
	DatabaseURL         string
	DBMaxConns          int
	DBMinConns          int
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	DBHealthCheckPeriod time.Duration
	DBConnectTimeout    time.Duration

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

	// Firebase Config
	FirebaseProjectID       string
	FirebaseCredentialsFile string
}

func LoadConfig() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		JWTSecret: getEnv("JWT_SECRET", "super-secret-key-bikko-dev"),

		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "bikko"),
		DBPassword:          getEnv("DB_PASSWORD", "bikkopass"),
		DBName:              getEnv("DB_NAME", "bikkodb"),
		DBSchema:            getEnv("DB_SCHEMA", "public"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		DBMaxConns:          getEnvInt("DB_MAX_CONNS", 25),
		DBMinConns:          getEnvInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime:   getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime:   getEnvDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBHealthCheckPeriod: getEnvDuration("DB_HEALTH_CHECK_PERIOD", 1*time.Minute),
		DBConnectTimeout:    getEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		R2AccountID:  getEnv("R2_ACCOUNT_ID", "mock_r2_account"),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", "mock_r2_access_key"),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", "mock_r2_secret_key"),
		R2BucketName: getEnv("R2_BUCKET_NAME", "bikko-media"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", "https://pub-bikko.r2.dev"),

		FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", "dummy-project"),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", ""),
	}
}

func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
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

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if n, err := strconv.Atoi(value); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if d, err := time.ParseDuration(value); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

