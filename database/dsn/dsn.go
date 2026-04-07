// Package dsn builds database connection strings from structured config or environment variables.
package dsn

import (
	"fmt"
	"net/url"
	"os"
)

// Config holds the fields needed to build a database connection string.
type Config struct {
	URL      string // Full connection string; takes precedence over individual fields.
	Host     string
	Port     string
	User     string
	Password string
	Name     string // Database name.
	Adapter  string // "postgresql", "mysql", "sqlite".
}

// FromEnv reads database configuration from DATABASE_* environment variables.
func FromEnv() Config {
	return Config{
		URL:      os.Getenv("DATABASE_URL"),
		Host:     os.Getenv("DATABASE_HOST"),
		Port:     os.Getenv("DATABASE_PORT"),
		User:     os.Getenv("DATABASE_USER"),
		Password: os.Getenv("DATABASE_PASSWORD"),
		Name:     os.Getenv("DATABASE_NAME"),
		Adapter:  os.Getenv("DATABASE_ADAPTER"),
	}
}

// WithEnvFallback returns a copy of c with empty fields filled from DATABASE_* environment variables.
func (c Config) WithEnvFallback() Config {
	env := FromEnv()
	if c.URL == "" {
		c.URL = env.URL
	}
	if c.Host == "" {
		c.Host = env.Host
	}
	if c.Port == "" {
		c.Port = env.Port
	}
	if c.User == "" {
		c.User = env.User
	}
	if c.Password == "" {
		c.Password = env.Password
	}
	if c.Name == "" {
		c.Name = env.Name
	}
	if c.Adapter == "" {
		c.Adapter = env.Adapter
	}
	return c
}

// Build returns a connection string. If URL is set, it is returned as-is.
// Otherwise, a DSN is constructed from the individual fields based on Adapter.
func Build(c Config) string {
	if c.URL != "" {
		return c.URL
	}

	switch c.Adapter {
	case "postgresql":
		return buildPostgres(c)
	case "mysql":
		return buildMySQL(c)
	case "sqlite":
		return buildSQLite(c)
	default:
		// Default to postgres format.
		return buildPostgres(c)
	}
}

func buildPostgres(c Config) string {
	host := cmp(c.Host, "localhost")
	port := cmp(c.Port, "5432")
	user := cmp(c.User, "postgres")
	name := cmp(c.Name, "postgres")

	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	if c.Password != "" {
		u.User = url.UserPassword(user, c.Password)
	} else {
		u.User = url.User(user)
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func buildMySQL(c Config) string {
	host := cmp(c.Host, "127.0.0.1")
	port := cmp(c.Port, "3306")
	user := cmp(c.User, "root")
	name := cmp(c.Name, "mysql")

	// user:pass@tcp(host:port)/dbname?parseTime=True
	auth := user
	if c.Password != "" {
		auth = user + ":" + c.Password
	}
	return fmt.Sprintf("%s@tcp(%s:%s)/%s?parseTime=True", auth, host, port, name)
}

func buildSQLite(c Config) string {
	name := cmp(c.Name, "dev.db")
	return name
}

func cmp(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}
