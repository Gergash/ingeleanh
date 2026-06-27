package chain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E002_FailoverURLFromChainHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	primary := srv.URL
	hash := ParseEndpointHash(primary)
	cache := NewCache()
	cache.Update(hash, 30, 1, 0)
	reader, err := NewReader("", "", cache)
	require.NoError(t, err)

	matches := MatchingURLs(hash, []string{"http://wrong:9999", primary, "http://other:8443"})
	require.Equal(t, []string{primary}, matches)

	url, cfg, err := ResolveVerifiedURL(context.Background(), reader, []string{"http://wrong:9999", primary})
	require.NoError(t, err)
	require.Equal(t, primary, url)
	require.Equal(t, uint64(1), cfg.Version)
}

func TestMatchingURLs_NoMatch(t *testing.T) {
	out := MatchingURLs("0xabc", []string{"http://localhost:8443"})
	require.Empty(t, out)
}
