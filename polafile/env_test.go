package polafile

import (
	"testing"
)

func TestApplyEnvOverrides_NilPolafile(t *testing.T) {
	t.Setenv("POLA_CSRF_ENABLED", "true")
	pf := ApplyEnvOverrides(nil)
	if pf == nil {
		t.Fatal("expected non-nil Polafile")
	}
	if pf.CSRF == nil {
		t.Fatal("expected CSRF block to be created")
	}
	if pf.CSRF.Enabled == nil || !*pf.CSRF.Enabled {
		t.Fatal("expected CSRF.Enabled = true")
	}
}

func TestApplyEnvOverrides_OverridesExisting(t *testing.T) {
	pf := &Polafile{
		Database: &Database{Host: "localhost", Port: "5432"},
	}
	t.Setenv("POLA_DATABASE_HOST", "prod-db.example.com")

	pf = ApplyEnvOverrides(pf)
	if pf.Database.Host != "prod-db.example.com" {
		t.Fatalf("expected host override, got %q", pf.Database.Host)
	}
	if pf.Database.Port != "5432" {
		t.Fatalf("expected port unchanged, got %q", pf.Database.Port)
	}
}

func TestApplyEnvOverrides_BoolPtrFalse(t *testing.T) {
	pf := &Polafile{
		CSRF: &CSRF{Enabled: BoolPtr(true)},
	}
	t.Setenv("POLA_CSRF_ENABLED", "false")

	pf = ApplyEnvOverrides(pf)
	if pf.CSRF.Enabled == nil || *pf.CSRF.Enabled {
		t.Fatal("expected CSRF.Enabled = false")
	}
}

func TestApplyEnvOverrides_CreatesBlockOnEnvVar(t *testing.T) {
	pf := &Polafile{}
	t.Setenv("POLA_SESSION_ENABLED", "true")
	t.Setenv("POLA_SESSION_STORE", "redis")

	pf = ApplyEnvOverrides(pf)
	if pf.Session == nil {
		t.Fatal("expected Session block created")
	}
	if pf.Session.Enabled == nil || !*pf.Session.Enabled {
		t.Fatal("expected Session.Enabled = true")
	}
	if pf.Session.Store != "redis" {
		t.Fatalf("expected Session.Store = redis, got %q", pf.Session.Store)
	}
}

func TestApplyEnvOverrides_NoEnvNoChange(t *testing.T) {
	pf := &Polafile{
		Cache: &Cache{Adapter: "memory"},
	}

	pf = ApplyEnvOverrides(pf)
	if pf.Cache.Adapter != "memory" {
		t.Fatalf("expected adapter unchanged, got %q", pf.Cache.Adapter)
	}
	if pf.Session != nil {
		t.Fatal("expected Session to remain nil")
	}
}

func TestApplyEnvOverrides_IntAndFloat(t *testing.T) {
	pf := &Polafile{}
	t.Setenv("POLA_RATE_LIMIT_ENABLED", "true")
	t.Setenv("POLA_RATE_LIMIT_RPS", "50.5")
	t.Setenv("POLA_RATE_LIMIT_BURST", "100")

	pf = ApplyEnvOverrides(pf)
	if pf.RateLimit == nil {
		t.Fatal("expected RateLimit block created")
	}
	if pf.RateLimit.RequestsPerSecond != 50.5 {
		t.Fatalf("expected RPS=50.5, got %f", pf.RateLimit.RequestsPerSecond)
	}
	if pf.RateLimit.Burst != 100 {
		t.Fatalf("expected Burst=100, got %d", pf.RateLimit.Burst)
	}
}

func TestApplyEnvOverrides_AllBlocks(t *testing.T) {
	t.Setenv("POLA_CSRF_ENABLED", "false")
	t.Setenv("POLA_SECURITY_HEADERS_ENABLED", "false")
	t.Setenv("POLA_CACHE_ADAPTER", "redis")
	t.Setenv("POLA_DATABASE_URL", "postgres://prod/db")
	t.Setenv("POLA_STORAGE_DRIVER", "rclone")
	t.Setenv("POLA_MAILER_TRANSPORT", "smtp")
	t.Setenv("POLA_IMAGE_PROCESSING_ENABLED", "true")
	t.Setenv("POLA_MCP_ENABLED", "true")
	t.Setenv("POLA_SESSION_STORE", "redis")
	t.Setenv("POLA_RATE_LIMIT_BURST", "50")
	t.Setenv("POLA_FLASH_ENABLED", "true")
	t.Setenv("POLA_I18N_DEFAULT_LOCALE", "fr")

	pf := ApplyEnvOverrides(nil)

	if pf.CSRF == nil || pf.CSRF.Enabled == nil || *pf.CSRF.Enabled {
		t.Error("CSRF should be disabled")
	}
	if pf.SecurityHeaders == nil || pf.SecurityHeaders.Enabled == nil || *pf.SecurityHeaders.Enabled {
		t.Error("SecurityHeaders should be disabled")
	}
	if pf.Cache == nil || pf.Cache.Adapter != "redis" {
		t.Error("Cache adapter should be redis")
	}
	if pf.Database == nil || pf.Database.URL != "postgres://prod/db" {
		t.Error("Database URL should be set")
	}
	if pf.Storage == nil || pf.Storage.Driver != "rclone" {
		t.Error("Storage driver should be rclone")
	}
	if pf.Mailer == nil || pf.Mailer.Transport != "smtp" {
		t.Error("Mailer transport should be smtp")
	}
	if pf.ImageProcessing == nil || pf.ImageProcessing.Enabled == nil || !*pf.ImageProcessing.Enabled {
		t.Error("ImageProcessing should be enabled")
	}
	if pf.MCP == nil || pf.MCP.Enabled == nil || !*pf.MCP.Enabled {
		t.Error("MCP should be enabled")
	}
	if pf.Session == nil || pf.Session.Store != "redis" {
		t.Error("Session store should be redis")
	}
	if pf.RateLimit == nil || pf.RateLimit.Burst != 50 {
		t.Error("RateLimit burst should be 50")
	}
	if pf.Flash == nil || pf.Flash.Enabled == nil || !*pf.Flash.Enabled {
		t.Error("Flash should be enabled")
	}
	if pf.I18n == nil || pf.I18n.DefaultLocale != "fr" {
		t.Error("I18n locale should be fr")
	}
}
