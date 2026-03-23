package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDotEnvFile(t *testing.T) {
	clearConfigEnv(t)

	dir := t.TempDir()
	chdir(t, dir)

	env := "" +
		"CLOUDFLARE_API_TOKEN=test-token\n" +
		"CLOUDFLARE_ZONE_NAME=example.com\n" +
		"CLOUDFLARE_RECORD_NAME=home.example.com\n" +
		"HTTP_USER_AGENT=cfddns/1.0 (+https://github.com/maxnguyen95/cfddns)\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CloudflareToken != "test-token" {
		t.Fatalf("CloudflareToken = %q, want %q", cfg.CloudflareToken, "test-token")
	}
	if cfg.ZoneName != "example.com" {
		t.Fatalf("ZoneName = %q, want %q", cfg.ZoneName, "example.com")
	}
	if cfg.RecordName != "home.example.com" {
		t.Fatalf("RecordName = %q, want %q", cfg.RecordName, "home.example.com")
	}
	if cfg.UserAgent != "cfddns/1.0 (+https://github.com/maxnguyen95/cfddns)" {
		t.Fatalf("UserAgent = %q", cfg.UserAgent)
	}
}

func TestLoadPrefersEnvironmentOverDotEnv(t *testing.T) {
	clearConfigEnv(t)

	dir := t.TempDir()
	chdir(t, dir)

	env := "" +
		"CLOUDFLARE_API_TOKEN=file-token\n" +
		"CLOUDFLARE_ZONE_NAME=file.example.com\n" +
		"CLOUDFLARE_RECORD_NAME=home.file.example.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := os.Setenv("CLOUDFLARE_ZONE_NAME", "env.example.com"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	if err := os.Setenv("CLOUDFLARE_RECORD_NAME", "home.env.example.com"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("CLOUDFLARE_ZONE_NAME")
		_ = os.Unsetenv("CLOUDFLARE_RECORD_NAME")
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ZoneName != "env.example.com" {
		t.Fatalf("ZoneName = %q, want %q", cfg.ZoneName, "env.example.com")
	}
	if cfg.CloudflareToken != "file-token" {
		t.Fatalf("CloudflareToken = %q, want %q", cfg.CloudflareToken, "file-token")
	}
	if cfg.RecordName != "home.env.example.com" {
		t.Fatalf("RecordName = %q, want %q", cfg.RecordName, "home.env.example.com")
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	envs := []string{
		"CLOUDFLARE_API_TOKEN",
		"CLOUDFLARE_ZONE_NAME",
		"CLOUDFLARE_RECORD_NAME",
		"CLOUDFLARE_RECORD_TYPE",
		"CLOUDFLARE_RECORD_PROXIED",
		"CLOUDFLARE_RECORD_TTL",
		"CLOUDFLARE_RECORD_COMMENT",
		"SYNC_INTERVAL",
		"HTTP_TIMEOUT",
		"HTTP_USER_AGENT",
	}

	original := make(map[string]*string, len(envs))
	for _, key := range envs {
		if value, ok := os.LookupEnv(key); ok {
			valueCopy := value
			original[key] = &valueCopy
		} else {
			original[key] = nil
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, key := range envs {
			value := original[key]
			var err error
			if value == nil {
				err = os.Unsetenv(key)
			} else {
				err = os.Setenv(key, *value)
			}
			if err != nil {
				t.Fatalf("restore %s: %v", key, err)
			}
		}
	})
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}
