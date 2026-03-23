package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	CloudflareToken string
	ZoneName        string
	RecordName      string
	RecordType      string
	Proxied         *bool
	TTL             *int
	Comment         string
	SyncInterval    time.Duration
	HTTPTimeout     time.Duration
	UserAgent       string
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		CloudflareToken: strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN")),
		ZoneName:        normalizeFQDN(os.Getenv("CLOUDFLARE_ZONE_NAME")),
		RecordName:      normalizeFQDN(os.Getenv("CLOUDFLARE_RECORD_NAME")),
		RecordType:      strings.ToUpper(strings.TrimSpace(defaultString(os.Getenv("CLOUDFLARE_RECORD_TYPE"), "A"))),
		Comment:         strings.TrimSpace(os.Getenv("CLOUDFLARE_RECORD_COMMENT")),
		UserAgent:       strings.TrimSpace(defaultString(os.Getenv("HTTP_USER_AGENT"), "cfddns/1.0 (+https://github.com/maxnguyen95/cfddns)")),
	}

	interval, err := parseDurationWithDefault("SYNC_INTERVAL", "5m")
	if err != nil {
		return Config{}, err
	}
	cfg.SyncInterval = interval

	httpTimeout, err := parseDurationWithDefault("HTTP_TIMEOUT", "10s")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPTimeout = httpTimeout

	proxied, err := parseOptionalBool("CLOUDFLARE_RECORD_PROXIED")
	if err != nil {
		return Config{}, err
	}
	cfg.Proxied = proxied

	ttl, err := parseOptionalTTL("CLOUDFLARE_RECORD_TTL")
	if err != nil {
		return Config{}, err
	}
	cfg.TTL = ttl

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	if c.CloudflareToken == "" {
		missing = append(missing, "CLOUDFLARE_API_TOKEN")
	}
	if c.ZoneName == "" {
		missing = append(missing, "CLOUDFLARE_ZONE_NAME")
	}
	if c.RecordName == "" {
		missing = append(missing, "CLOUDFLARE_RECORD_NAME")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	switch c.RecordType {
	case "A", "AAAA":
	default:
		return fmt.Errorf("unsupported CLOUDFLARE_RECORD_TYPE %q: only A and AAAA are supported", c.RecordType)
	}

	if !strings.HasSuffix(c.RecordName, "."+c.ZoneName) && c.RecordName != c.ZoneName {
		return errors.New("CLOUDFLARE_RECORD_NAME must be the zone apex or a subdomain of CLOUDFLARE_ZONE_NAME")
	}

	return nil
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Clean(path), err)
	}

	for i, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("invalid %s line %d: expected KEY=VALUE", filepath.Clean(path), i+1)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("invalid %s line %d: empty key", filepath.Clean(path), i+1)
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s: %w", key, filepath.Clean(path), err)
		}
	}

	return nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func normalizeFQDN(v string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(v)), ".")
}

func parseDurationWithDefault(envName, fallback string) (time.Duration, error) {
	value := defaultString(os.Getenv(envName), fallback)
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", envName, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid %s: must be > 0", envName)
	}
	return d, nil
}

func parseOptionalBool(envName string) (*bool, error) {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envName, err)
	}
	return &parsed, nil
}

func parseOptionalTTL(envName string) (*int, error) {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envName, err)
	}
	if parsed != 1 && (parsed < 60 || parsed > 86400) {
		return nil, fmt.Errorf("invalid %s: must be 1 (automatic) or between 60 and 86400", envName)
	}
	return &parsed, nil
}
