package camouflage

import (
	"math/rand"
	"strings"
	"time"
)

const DefaultUserAgent = "ResidentialHub/1.0 (gateway)"

func BeaconInterval(baseSec int, jitterPercent int) time.Duration {
	if jitterPercent <= 0 {
		return time.Duration(baseSec) * time.Second
	}
	jitter := float64(baseSec) * float64(jitterPercent) / 100.0
	delta := (rand.Float64()*2 - 1) * jitter
	sec := float64(baseSec) + delta
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec * float64(time.Second))
}

func IoTHeaders() map[string]string {
	return map[string]string{
		"User-Agent":        DefaultUserAgent,
		"X-Client-Type":     "iot-gateway",
		"X-Device-Firmware": "2.1.0-lab",
	}
}

func SanitizeLog(s string) string {
	forbidden := []string{"whoami", "cmd.exe", "/bin/sh"}
	out := s
	for _, f := range forbidden {
		out = strings.ReplaceAll(out, f, "[redacted]")
	}
	return out
}
