package publicip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

type Provider struct {
	httpClient *http.Client
	userAgent  string
}

func New(httpClient *http.Client, userAgent string) *Provider {
	return &Provider{httpClient: httpClient, userAgent: userAgent}
}

func (p *Provider) Detect(ctx context.Context, recordType string) (string, error) {
	endpoints := endpointsFor(recordType)
	var errs []error

	for _, endpoint := range endpoints {
		ip, err := p.fetch(ctx, endpoint, recordType)
		if err == nil {
			return ip, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
	}

	return "", errors.Join(errs...)
}

func endpointsFor(recordType string) []string {
	switch recordType {
	case "AAAA":
		return []string{
			"https://ipv6.icanhazip.com",
			"https://ifconfig.co/ip",
		}
	default:
		return []string{
			"https://ipv4.icanhazip.com",
			"https://api.ipify.org",
			"https://ifconfig.me/ip",
		}
	}
}

func (p *Provider) fetch(ctx context.Context, endpoint string, recordType string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	if p.userAgent != "" {
		request.Header.Set("User-Agent", p.userAgent)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status %s", response.Status)
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, 128))
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(payload))
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid IP %q", ip)
	}

	if recordType == "AAAA" && parsed.To4() != nil {
		return "", fmt.Errorf("provider returned IPv4 %q for AAAA record", ip)
	}
	if recordType == "A" && parsed.To4() == nil {
		return "", fmt.Errorf("provider returned non-IPv4 %q for A record", ip)
	}

	return ip, nil
}
