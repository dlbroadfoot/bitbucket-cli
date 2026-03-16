package keyring

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	envKeyringTimeout = "BB_KEYRING_TIMEOUT"

	timeoutHeadless    = 3 * time.Second
	timeoutInteractive = 30 * time.Second
)

// IsHeadless returns true if the environment is likely unable to handle keyring
// unlock prompts without hanging (SSH without display, containers, CI, cron).
func IsHeadless() bool {
	isSSH := os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_CONNECTION") != ""
	if isSSH {
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			return true
		}
		hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
		return !hasDisplay
	}

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return envTruthy(os.Getenv("CI"))
	}

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	hasDBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS") != ""
	return !hasDisplay && !hasDBus
}

// Timeout returns the appropriate keyring timeout for the current environment.
// It checks BB_KEYRING_TIMEOUT env var first, then uses 3s for headless or 30s for interactive.
func Timeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv(envKeyringTimeout)); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	if IsHeadless() {
		return timeoutHeadless
	}
	return timeoutInteractive
}

func envTruthy(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "1" || val == "true" || val == "yes"
}
