package agent

import (
	"context"
	"log"
	"strings"

	"github.com/ingeleanh/c2-blockchain/internal/chain"
)

// Failover resolves a verified C2 URL from on-chain endpointHash (MVP-05 / DEMO-006).
type Failover struct {
	Reader     *chain.Reader
	Candidates []string
	CurrentURL string
}

func (f *Failover) Attempt(ctx context.Context) (string, chain.Config, bool) {
	if f.Reader == nil || len(f.Candidates) == 0 {
		return "", chain.Config{}, false
	}
	url, cfg, err := chain.ResolveVerifiedURL(ctx, f.Reader, f.Candidates)
	if err != nil {
		log.Printf("failover: getConfig: %v", err)
		return "", chain.Config{}, false
	}
	if url == "" || url == strings.TrimRight(f.CurrentURL, "/") {
		return "", cfg, false
	}
	return url, cfg, true
}

// ParseURLCandidates builds ordered unique URL list from primary + env extras.
func ParseURLCandidates(primary string, extras string) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(strings.TrimRight(u, "/"))
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(primary)
	for _, part := range strings.Split(extras, ",") {
		add(part)
	}
	return out
}
