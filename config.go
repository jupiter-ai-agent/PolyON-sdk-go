// Package polyon provides the PolyON Platform SDK for Go modules.
//
// It automatically loads PRC-injected environment variables into typed config structs.
package polyon

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all PRC-provisioned resource configurations.
type Config struct {
	Auth      AuthConfig
	Database  DatabaseConfig
	Storage   StorageConfig
	Cache     CacheConfig
	Search    SearchConfig
	Directory DirectoryConfig
	SMTP      SMTPConfig
}

// AuthConfig holds OIDC authentication settings (PRC auth provider).
type AuthConfig struct {
	Issuer                string // OIDC Issuer URL (external)
	ClientID              string // OIDC Client ID
	ClientSecret          string // OIDC Client Secret (confidential only)
	AuthEndpoint          string // Authorization endpoint (external, browser redirect)
	TokenEndpoint         string // Token endpoint (internal, server-to-server)
	TokenEndpointExternal string // Token endpoint (external, for reference)
	JWKSURI               string // JWKS endpoint (internal, server-to-server)
	JWKSURIExternal       string // JWKS endpoint (external, for reference)
}

// DatabaseConfig holds PostgreSQL connection settings (PRC database provider).
type DatabaseConfig struct {
	URL      string // Full connection URL: postgres://user:pass@host:port/db
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// StorageConfig holds S3-compatible storage settings (PRC objectStorage provider).
type StorageConfig struct {
	Endpoint  string // S3 API endpoint (e.g., http://polyon-rustfs:9000)
	Bucket    string
	AccessKey string
	SecretKey string
}

// CacheConfig holds Redis connection settings (PRC cache provider).
type CacheConfig struct {
	Host string
	Port int
	DB   int
}

// SearchConfig holds OpenSearch settings (PRC search provider).
type SearchConfig struct {
	Endpoint    string // e.g., http://polyon-search:9200
	IndexPrefix string
}

// DirectoryConfig holds LDAP settings (PRC directory provider).
type DirectoryConfig struct {
	Host         string
	Port         int
	BaseDN       string
	BindDN       string
	BindPassword string
}

// SMTPConfig holds mail settings (PRC smtp provider).
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// Load reads PRC environment variables and returns a Config.
// Missing variables are left as zero values — only claimed resources will be populated.
func Load() (*Config, error) {
	cfg := &Config{}

	// Auth
	cfg.Auth = AuthConfig{
		Issuer:                env("OIDC_ISSUER"),
		ClientID:              env("OIDC_CLIENT_ID"),
		ClientSecret:          env("OIDC_CLIENT_SECRET"),
		AuthEndpoint:          env("OIDC_AUTH_ENDPOINT"),
		TokenEndpoint:         env("OIDC_TOKEN_ENDPOINT"),
		TokenEndpointExternal: env("OIDC_TOKEN_ENDPOINT_EXTERNAL"),
		JWKSURI:               env("OIDC_JWKS_URI"),
		JWKSURIExternal:       env("OIDC_JWKS_URI_EXTERNAL"),
	}

	// Database
	cfg.Database = DatabaseConfig{
		URL:      env("DATABASE_URL"),
		Host:     envOr("DB_HOST", envOr("PGHOST", "")),
		Port:     envInt("DB_PORT", 5432),
		Database: envOr("DB_NAME", envOr("PGDATABASE", "")),
		User:     envOr("DB_USER", envOr("PGUSER", "")),
		Password: envOr("DB_PASSWORD", envOr("PGPASSWORD", "")),
	}

	// Storage (S3)
	cfg.Storage = StorageConfig{
		Endpoint:  env("S3_ENDPOINT"),
		Bucket:    env("S3_BUCKET"),
		AccessKey: envOr("S3_ACCESS_KEY", env("AWS_ACCESS_KEY_ID")),
		SecretKey: envOr("S3_SECRET_KEY", env("AWS_SECRET_ACCESS_KEY")),
	}

	// Cache (Redis)
	cfg.Cache = CacheConfig{
		Host: envOr("REDIS_HOST", ""),
		Port: envInt("REDIS_PORT", 6379),
		DB:   envInt("REDIS_DB", 0),
	}

	// Search (OpenSearch)
	cfg.Search = SearchConfig{
		Endpoint:    env("SEARCH_ENDPOINT"),
		IndexPrefix: env("SEARCH_INDEX_PREFIX"),
	}

	// Directory (LDAP)
	cfg.Directory = DirectoryConfig{
		Host:         env("LDAP_HOST"),
		Port:         envInt("LDAP_PORT", 389),
		BaseDN:       env("LDAP_BASE_DN"),
		BindDN:       env("LDAP_BIND_DN"),
		BindPassword: env("LDAP_BIND_PASSWORD"),
	}

	// SMTP
	cfg.SMTP = SMTPConfig{
		Host:     env("SMTP_HOST"),
		Port:     envInt("SMTP_PORT", 587),
		User:     env("SMTP_USER"),
		Password: env("SMTP_PASSWORD"),
	}

	return cfg, nil
}

// MustLoad calls Load and panics on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("polyon: config load failed: %v", err))
	}
	return cfg
}

// HasAuth returns true if OIDC auth is configured.
func (c *Config) HasAuth() bool { return c.Auth.Issuer != "" && c.Auth.ClientID != "" }

// HasDatabase returns true if database is configured.
func (c *Config) HasDatabase() bool { return c.Database.URL != "" || c.Database.Host != "" }

// HasStorage returns true if S3 storage is configured.
func (c *Config) HasStorage() bool { return c.Storage.Endpoint != "" }

// HasCache returns true if Redis cache is configured.
func (c *Config) HasCache() bool { return c.Cache.Host != "" }

// HasSearch returns true if OpenSearch is configured.
func (c *Config) HasSearch() bool { return c.Search.Endpoint != "" }

// HasDirectory returns true if LDAP directory is configured.
func (c *Config) HasDirectory() bool { return c.Directory.Host != "" }

// HasSMTP returns true if SMTP is configured.
func (c *Config) HasSMTP() bool { return c.SMTP.Host != "" }

func env(key string) string {
	return os.Getenv(key)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
