package chain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCHAIN001_IndexerParseConfigUpdated(t *testing.T) {
	c := NewCache()
	c.IndexConfigUpdated("0xabc", 30, 1, 42018)
	cfg, block := c.Get()
	require.Equal(t, "0xabc", cfg.EndpointHash)
	require.Equal(t, uint32(30), cfg.BeaconIntervalSec)
	require.Equal(t, uint64(1), cfg.Version)
	require.Equal(t, uint64(42018), block)
}

func TestCHAIN002_GetConfigFromCache(t *testing.T) {
	c := NewCache()
	c.Update("0xdead", 60, 2, 100)
	r := &Reader{cache: c}
	cfg, err := r.GetConfig(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(2), cfg.Version)
	require.Equal(t, uint32(60), cfg.BeaconIntervalSec)
}
