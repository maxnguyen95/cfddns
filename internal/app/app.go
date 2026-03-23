package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/maxnguyen95/cfddns/internal/cloudflare"
	"github.com/maxnguyen95/cfddns/internal/config"
	"github.com/maxnguyen95/cfddns/internal/ddns"
	"github.com/maxnguyen95/cfddns/internal/publicip"
)

func Run(ctx context.Context, logger *slog.Logger, once bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	cfClient := cloudflare.New(httpClient, cfg.CloudflareToken, cfg.UserAgent)
	ipProvider := publicip.New(httpClient, cfg.UserAgent)
	service := ddns.New(cfg, logger, cfClient, ipProvider)

	if once {
		_, err := service.Sync(ctx)
		return err
	}

	logger.Info("starting DDNS sync loop", slog.String("zone", cfg.ZoneName), slog.String("record_name", cfg.RecordName), slog.String("record_type", cfg.RecordType), slog.Duration("sync_interval", cfg.SyncInterval))
	if _, err := service.Sync(ctx); err != nil {
		logger.Error("initial sync failed", slog.String("error", err.Error()))
	}

	ticker := time.NewTicker(cfg.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down DDNS sync loop")
			return nil
		case <-ticker.C:
			if _, err := service.Sync(ctx); err != nil {
				logger.Error("sync failed", slog.String("error", err.Error()))
			}
		}
	}
}
