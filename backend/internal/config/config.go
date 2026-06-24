package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully-resolved service configuration, loaded once at startup.
type Config struct {
	Port        string
	DatabaseURL string
	CORSOrigins []string
	AppBaseURL  string // base for YooKassa return_url, e.g. http://localhost:5173

	S3 S3Config
	YK YooKassaConfig

	ClamAVAddr       string        // host:port of clamd; empty disables scanning
	SwaggerEnabled   bool          // serve /swagger in dev
	TrendingInterval time.Duration // how often to recompute trending scores
}

// S3Config configures the object-storage client (MinIO locally, S3/Yandex in prod).
type S3Config struct {
	Endpoint  string // in-network endpoint the API uses to talk to storage, e.g. http://minio:9000
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	// PublicEndpoint is the address a BROWSER can reach storage at — it must be
	// host-mapped (e.g. http://localhost:9000), not the in-network service
	// name, and must NOT include the bucket. It is used both to build public
	// object URLs and to presign downloads, since a presigned URL's signature
	// covers the Host header and so must be signed for the same host the
	// browser will actually request.
	PublicEndpoint string
}

// YooKassaConfig holds the merchant credentials for the YooKassa client. An
// empty ShopID/SecretKey is valid — payment creation will simply fail with a
// clear error until they're configured.
type YooKassaConfig struct {
	ShopID    string
	SecretKey string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: env("DATABASE_URL", ""),
		CORSOrigins: splitList(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		AppBaseURL:  env("APP_BASE_URL", "http://localhost:5173"),
		S3: S3Config{
			Endpoint:       env("S3_ENDPOINT", "http://localhost:9000"),
			Region:         env("S3_REGION", "us-east-1"),
			Bucket:         env("S3_BUCKET", "indieforge"),
			AccessKey:      env("S3_ACCESS_KEY", ""),
			SecretKey:      env("S3_SECRET_KEY", ""),
			PublicEndpoint: env("S3_PUBLIC_ENDPOINT", "http://localhost:9000"),
		},
		YK: YooKassaConfig{
			ShopID:    env("YOOKASSA_SHOP_ID", ""),
			SecretKey: env("YOOKASSA_SECRET_KEY", ""),
		},
		ClamAVAddr:       env("CLAMAV_ADDR", ""),
		SwaggerEnabled:   envBool("SWAGGER_ENABLED", true),
		TrendingInterval: envDuration("TRENDING_INTERVAL", 15*time.Minute),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
