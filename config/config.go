package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port string
	Env  string

	JWTSecret         string
	LocalTestAccounts bool
	LocalStorage      bool
	LocalStorageDir   string

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
		Env:  getEnv("ENV", ""),

		JWTSecret:         getEnv("JWT_SECRET", ""),
		LocalTestAccounts: getEnv("LOCAL_TEST_ACCOUNTS", "false") == "true",
		LocalStorage:      getEnv("LOCAL_STORAGE", "false") == "true",
		LocalStorageDir:   getEnv("LOCAL_STORAGE_DIR", ".local-media"),

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

		R2AccountID:  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName: getEnv("R2_BUCKET_NAME", "bikko-media"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", ""),

		FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", ""),
	}
}

// Validate is called before opening dependencies or serving traffic. Test capabilities
// require two explicit settings and cannot be enabled by missing configuration.
func (c *Config) Validate() error {
	if c.Env != "development" && c.Env != "test" && c.Env != "production" {
		return fmt.Errorf("ENV must be development, test or production")
	}
	if len(c.JWTSecret) < 32 || strings.Contains(c.JWTSecret, "super-secret-key-bikko") {
		return fmt.Errorf("JWT_SECRET must contain at least 32 characters and cannot use the legacy default")
	}
	if (c.LocalTestAccounts || c.LocalStorage) && c.Env != "development" {
		return fmt.Errorf("local test accounts and local storage require ENV=development")
	}
	if !c.LocalTestAccounts && c.FirebaseProjectID == "" {
		return fmt.Errorf("FIREBASE_PROJECT_ID is required outside explicit local account mode")
	}
	if !c.LocalStorage {
		for _, value := range []string{c.R2AccountID, c.R2AccessKey, c.R2SecretKey} {
			if value == "" || strings.HasPrefix(value, "mock_") {
				return fmt.Errorf("R2 credentials are required unless LOCAL_STORAGE=true")
			}
		}
		u, err := url.Parse(c.R2PublicURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("R2_PUBLIC_URL must be an HTTPS URL")
		}
	}
	return nil
}

func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(c.DBUser, c.DBPassword), Host: net.JoinHostPort(c.DBHost, c.DBPort), Path: "/" + c.DBName}
	q := url.Values{"sslmode": {c.DBSSLMode}}
	u.RawQuery = q.Encode()
	return u.String()
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
