package executor

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func Shell(ctx context.Context, argv []string) (Result, error) {
	if len(argv) == 0 {
		return Result{ExitCode: 1, Stderr: "empty argv"}, nil
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", strings.Join(argv, " "))
	} else {
		cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
	}
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return Result{ExitCode: ee.ExitCode(), Stderr: string(ee.Stderr)}, nil
		}
		return Result{ExitCode: 1, Stderr: err.Error()}, nil
	}
	return Result{ExitCode: 0, Stdout: strings.TrimSpace(string(out))}, nil
}

func Whoami(ctx context.Context) (Result, error) {
	if runtime.GOOS == "windows" {
		return Shell(ctx, []string{"whoami"})
	}
	return Shell(ctx, []string{"whoami"})
}

func CurrentOS() string {
	if runtime.GOOS == "windows" {
		return "windows-amd64"
	}
	return "linux-amd64"
}

func Hostname() string {
	h, _ := os.Hostname()
	if h == "" {
		return "lab-host"
	}
	return h
}
