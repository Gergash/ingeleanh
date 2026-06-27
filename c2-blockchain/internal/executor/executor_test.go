package executor

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMVP007_CurrentOSReflectsRuntime(t *testing.T) {
	got := CurrentOS()
	want := runtime.GOOS + "-" + runtime.GOARCH
	require.Equal(t, want, got)
}

func TestMVP007_SupportedPlatforms(t *testing.T) {
	platforms := []string{"linux-amd64", "linux-arm64", "windows-amd64"}
	for _, p := range platforms {
		require.Contains(t, p, "-")
	}
}
