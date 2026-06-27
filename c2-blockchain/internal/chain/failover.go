package chain

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// MatchingURLs returns candidate URLs whose SHA-256 hash matches the on-chain endpointHash.
func MatchingURLs(endpointHash string, candidates []string) []string {
	want := NormalizeHash(endpointHash)
	var out []string
	for _, u := range candidates {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if NormalizeHash(ParseEndpointHash(u)) == want {
			out = append(out, strings.TrimRight(u, "/"))
		}
	}
	return out
}

// ResolveVerifiedURL picks the first candidate that matches chain config and responds healthy.
func ResolveVerifiedURL(ctx context.Context, reader *Reader, candidates []string) (string, Config, error) {
	cfg, err := reader.GetConfig(ctx)
	if err != nil {
		return "", Config{}, err
	}
	matches := MatchingURLs(cfg.EndpointHash, candidates)
	if len(matches) == 0 {
		return "", cfg, nil
	}
	for _, url := range matches {
		if HealthOK(ctx, url) {
			return url, cfg, nil
		}
	}
	return "", cfg, nil
}

// HealthOK checks C2 server health endpoint.
func HealthOK(ctx context.Context, baseURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/health", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
