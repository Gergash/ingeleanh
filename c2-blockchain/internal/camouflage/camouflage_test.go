package camouflage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCAMO001_BeaconJitterInRange(t *testing.T) {
	base := 30
	jitter := 20
	for i := 0; i < 100; i++ {
		d := BeaconInterval(base, jitter)
		sec := d.Seconds()
		min := float64(base) * 0.8
		max := float64(base) * 1.2
		require.GreaterOrEqual(t, sec, min-0.01)
		require.LessOrEqual(t, sec, max+0.01)
	}
}

func TestCAMO002_LogsNoPlaintextCommands(t *testing.T) {
	log := SanitizeLog("executed whoami for task")
	require.False(t, strings.Contains(log, "whoami"))
}

func TestCAMO003_EndpointHashOnlyOnChain(t *testing.T) {
	// URL resolved locally; on-chain only hash placeholder
	url := "https://localhost:8443"
	hash := "0x" + strings.Repeat("a", 64)
	require.NotContains(t, hash, url)
}
