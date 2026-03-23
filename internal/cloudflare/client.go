package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const baseURL = "https://api.cloudflare.com/client/v4"

type Client struct {
	httpClient *http.Client
	token      string
	userAgent  string
}

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied *bool  `json:"proxied,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type APIEnvelope[T any] struct {
	Success bool       `json:"success"`
	Errors  []APIError `json:"errors"`
	Result  T          `json:"result"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type DNSRecordRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     *int   `json:"ttl,omitempty"`
	Proxied *bool  `json:"proxied,omitempty"`
	Comment string `json:"comment,omitempty"`
}

func New(httpClient *http.Client, token string, userAgent string) *Client {
	return &Client{httpClient: httpClient, token: token, userAgent: userAgent}
}

func (c *Client) FindZoneByName(ctx context.Context, zoneName string) (Zone, error) {
	query := url.Values{}
	query.Set("name", zoneName)
	query.Set("status", "active")
	query.Set("match", "all")

	var envelope APIEnvelope[[]Zone]
	if err := c.doJSON(ctx, http.MethodGet, "/zones?"+query.Encode(), nil, &envelope); err != nil {
		return Zone{}, err
	}
	if len(envelope.Result) == 0 {
		return Zone{}, fmt.Errorf("cloudflare zone %q not found", zoneName)
	}
	return envelope.Result[0], nil
}

func (c *Client) FindDNSRecord(ctx context.Context, zoneID, recordType, recordName string) (*DNSRecord, error) {
	query := url.Values{}
	query.Set("type", recordType)
	query.Set("name", recordName)
	query.Set("match", "all")

	var envelope APIEnvelope[[]DNSRecord]
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/zones/%s/dns_records?%s", zoneID, query.Encode()), nil, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Result) == 0 {
		return nil, nil
	}
	return &envelope.Result[0], nil
}

func (c *Client) CreateDNSRecord(ctx context.Context, zoneID string, req DNSRecordRequest) (*DNSRecord, error) {
	var envelope APIEnvelope[DNSRecord]
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/zones/%s/dns_records", zoneID), req, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Result, nil
}

func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, req DNSRecordRequest) (*DNSRecord, error) {
	var envelope APIEnvelope[DNSRecord]
	if err := c.doJSON(ctx, http.MethodPatch, fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), req, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Result, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody interface{}, out interface{}) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call cloudflare api: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read cloudflare api response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("cloudflare api returned %s: %s", response.Status, strings.TrimSpace(string(payload)))
	}

	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode cloudflare api response: %w", err)
	}

	if message := firstAPIError(payload); message != "" {
		return fmt.Errorf("cloudflare api error: %s", message)
	}

	return nil
}

func firstAPIError(payload []byte) string {
	var generic struct {
		Success bool       `json:"success"`
		Errors  []APIError `json:"errors"`
	}
	if err := json.Unmarshal(payload, &generic); err != nil {
		return ""
	}
	if generic.Success || len(generic.Errors) == 0 {
		return ""
	}
	return generic.Errors[0].Message
}
