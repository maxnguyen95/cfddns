package ddns

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maxnguyen95/cfddns/internal/cloudflare"
	"github.com/maxnguyen95/cfddns/internal/config"
	"github.com/maxnguyen95/cfddns/internal/publicip"
)

type Service struct {
	cfg          config.Config
	logger       *slog.Logger
	cfClient     *cloudflare.Client
	publicIP     *publicip.Provider
	resolvedZone *cloudflare.Zone
}

type Result struct {
	Changed   bool
	Action    string
	RecordID  string
	CurrentIP string
}

func New(cfg config.Config, logger *slog.Logger, cfClient *cloudflare.Client, publicIP *publicip.Provider) *Service {
	return &Service{cfg: cfg, logger: logger, cfClient: cfClient, publicIP: publicIP}
}

func (s *Service) Sync(ctx context.Context) (Result, error) {
	zone, err := s.ensureZone(ctx)
	if err != nil {
		return Result{}, err
	}

	currentIP, err := s.publicIP.Detect(ctx, s.cfg.RecordType)
	if err != nil {
		return Result{}, fmt.Errorf("detect public IP: %w", err)
	}

	record, err := s.cfClient.FindDNSRecord(ctx, zone.ID, s.cfg.RecordType, s.cfg.RecordName)
	if err != nil {
		return Result{}, fmt.Errorf("find dns record: %w", err)
	}

	if record == nil {
		created, err := s.cfClient.CreateDNSRecord(ctx, zone.ID, buildCreateRequest(s.cfg, currentIP))
		if err != nil {
			return Result{}, fmt.Errorf("create dns record: %w", err)
		}
		s.logger.Info("dns record created", slog.String("record_name", created.Name), slog.String("record_type", created.Type), slog.String("ip", created.Content), slog.String("record_id", created.ID))
		return Result{Changed: true, Action: "created", RecordID: created.ID, CurrentIP: created.Content}, nil
	}

	if record.Content == currentIP && shouldKeepRecordSettings(record, s.cfg) {
		s.logger.Info("dns record already up to date", slog.String("record_name", record.Name), slog.String("record_type", record.Type), slog.String("ip", record.Content), slog.String("record_id", record.ID))
		return Result{Changed: false, Action: "noop", RecordID: record.ID, CurrentIP: record.Content}, nil
	}

	updated, err := s.cfClient.UpdateDNSRecord(ctx, zone.ID, record.ID, buildUpdateRequest(record, s.cfg, currentIP))
	if err != nil {
		return Result{}, fmt.Errorf("update dns record: %w", err)
	}

	s.logger.Info("dns record updated", slog.String("record_name", updated.Name), slog.String("record_type", updated.Type), slog.String("old_ip", record.Content), slog.String("new_ip", updated.Content), slog.String("record_id", updated.ID))
	return Result{Changed: true, Action: "updated", RecordID: updated.ID, CurrentIP: updated.Content}, nil
}

func (s *Service) ensureZone(ctx context.Context) (cloudflare.Zone, error) {
	if s.resolvedZone != nil {
		return *s.resolvedZone, nil
	}
	zone, err := s.cfClient.FindZoneByName(ctx, s.cfg.ZoneName)
	if err != nil {
		return cloudflare.Zone{}, fmt.Errorf("resolve zone: %w", err)
	}
	s.resolvedZone = &zone
	return zone, nil
}

func buildCreateRequest(cfg config.Config, currentIP string) cloudflare.DNSRecordRequest {
	return cloudflare.DNSRecordRequest{
		Type:    cfg.RecordType,
		Name:    cfg.RecordName,
		Content: currentIP,
		TTL:     firstNonNilInt(cfg.TTL, intPtr(1)),
		Proxied: firstNonNilBool(cfg.Proxied, boolPtr(false)),
		Comment: cfg.Comment,
	}
}

func buildUpdateRequest(existing *cloudflare.DNSRecord, cfg config.Config, currentIP string) cloudflare.DNSRecordRequest {
	return cloudflare.DNSRecordRequest{
		Type:    cfg.RecordType,
		Name:    cfg.RecordName,
		Content: currentIP,
		TTL:     firstNonNilInt(cfg.TTL, intPtr(existing.TTL)),
		Proxied: firstNonNilBool(cfg.Proxied, existing.Proxied),
		Comment: firstNonEmpty(cfg.Comment, existing.Comment),
	}
}

func shouldKeepRecordSettings(record *cloudflare.DNSRecord, cfg config.Config) bool {
	if cfg.TTL != nil && record.TTL != *cfg.TTL {
		return false
	}
	if cfg.Proxied != nil {
		if record.Proxied == nil || *record.Proxied != *cfg.Proxied {
			return false
		}
	}
	if cfg.Comment != "" && record.Comment != cfg.Comment {
		return false
	}
	return true
}

func firstNonNilInt(values ...*int) *int {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstNonNilBool(values ...*bool) *bool {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func intPtr(v int) *int   { return &v }
func boolPtr(v bool) *bool { return &v }
