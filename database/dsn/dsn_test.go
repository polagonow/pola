package dsn

import (
	"os"
	"testing"
)

func TestBuild_URLPassthrough(t *testing.T) {
	c := Config{URL: "postgres://custom:pass@myhost:9999/mydb"}
	got := Build(c)
	if got != c.URL {
		t.Errorf("expected URL passthrough, got %q", got)
	}
}

func TestBuild_Postgres(t *testing.T) {
	c := Config{
		Adapter:  "postgresql",
		Host:     "db.example.com",
		Port:     "5433",
		User:     "admin",
		Password: "secret",
		Name:     "appdb",
	}
	got := Build(c)
	want := "postgres://admin:secret@db.example.com:5433/appdb?sslmode=prefer"
	if got != want {
		t.Errorf("postgres DSN:\n got  %q\n want %q", got, want)
	}
}

func TestBuild_PostgresDefaults(t *testing.T) {
	c := Config{Adapter: "postgres"}
	got := Build(c)
	want := "postgres://postgres@localhost:5432/postgres?sslmode=prefer"
	if got != want {
		t.Errorf("postgres defaults:\n got  %q\n want %q", got, want)
	}
}

func TestBuild_MySQL(t *testing.T) {
	c := Config{
		Adapter:  "mysql",
		Host:     "db.example.com",
		Port:     "3307",
		User:     "admin",
		Password: "secret",
		Name:     "appdb",
	}
	got := Build(c)
	want := "admin:secret@tcp(db.example.com:3307)/appdb?parseTime=True&tls=preferred"
	if got != want {
		t.Errorf("mysql DSN:\n got  %q\n want %q", got, want)
	}
}

func TestBuild_MySQLDefaults(t *testing.T) {
	c := Config{Adapter: "mysql"}
	got := Build(c)
	want := "root@tcp(127.0.0.1:3306)/mysql?parseTime=True&tls=preferred"
	if got != want {
		t.Errorf("mysql defaults:\n got  %q\n want %q", got, want)
	}
}

func TestBuild_SQLite(t *testing.T) {
	c := Config{Adapter: "sqlite", Name: "test.db"}
	got := Build(c)
	if got != "test.db" {
		t.Errorf("sqlite DSN: got %q, want %q", got, "test.db")
	}
}

func TestBuild_SQLiteDefault(t *testing.T) {
	c := Config{Adapter: "sqlite"}
	got := Build(c)
	if got != "dev.db" {
		t.Errorf("sqlite default: got %q, want %q", got, "dev.db")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://fromenv")
	t.Setenv("DATABASE_HOST", "envhost")
	t.Setenv("DATABASE_PORT", "9999")
	t.Setenv("DATABASE_USER", "envuser")
	t.Setenv("DATABASE_PASSWORD", "envpass")
	t.Setenv("DATABASE_NAME", "envdb")
	t.Setenv("DATABASE_ADAPTER", "postgresql")

	c := FromEnv()
	if c.URL != "postgres://fromenv" {
		t.Errorf("URL: got %q", c.URL)
	}
	if c.Host != "envhost" {
		t.Errorf("Host: got %q", c.Host)
	}
	if c.Port != "9999" {
		t.Errorf("Port: got %q", c.Port)
	}
	if c.User != "envuser" {
		t.Errorf("User: got %q", c.User)
	}
	if c.Password != "envpass" {
		t.Errorf("Password: got %q", c.Password)
	}
	if c.Name != "envdb" {
		t.Errorf("Name: got %q", c.Name)
	}
	if c.Adapter != "postgresql" {
		t.Errorf("Adapter: got %q", c.Adapter)
	}
}

func TestWithEnvFallback(t *testing.T) {
	// Clear any existing env vars.
	os.Unsetenv("DATABASE_URL")
	t.Setenv("DATABASE_HOST", "envhost")
	t.Setenv("DATABASE_PORT", "8888")
	t.Setenv("DATABASE_ADAPTER", "mysql")

	c := Config{
		Host:    "explicit-host",
		Adapter: "postgresql",
	}
	got := c.WithEnvFallback()

	// Explicit values should be preserved.
	if got.Host != "explicit-host" {
		t.Errorf("Host should be preserved, got %q", got.Host)
	}
	if got.Adapter != "postgresql" {
		t.Errorf("Adapter should be preserved, got %q", got.Adapter)
	}
	// Empty fields should be filled from env.
	if got.Port != "8888" {
		t.Errorf("Port should fall back to env, got %q", got.Port)
	}
}
