package executor

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEXEC001_ShellLinuxArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("linux test")
	}
	res, err := Whoami(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NotEmpty(t, res.Stdout)
}

func TestEXEC002_ShellWindowsArgv(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows test")
	}
	res, err := Whoami(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
	require.NotEmpty(t, res.Stdout)
}

func TestEXEC003_MsfModuleOptional(t *testing.T) {
	// Bridge stub: msf_module returns simulated result via C2 channel
	res, err := Shell(context.Background(), []string{"echo", "msf-simulated"})
	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode)
}
