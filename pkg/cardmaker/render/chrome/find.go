package chrome

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Environment variables read by the commands.
const (
	// EnvBrowserPath points to the browser executable.
	EnvBrowserPath = "CARDMAKER_CHROME"

	// EnvNoSandbox turns on Options.NoSandbox; see NoSandboxFromEnv.
	EnvNoSandbox = "CARDMAKER_CHROME_NO_SANDBOX"
)

// NoSandboxFromEnv reports whether EnvNoSandbox is set to a true value
// ("1", "true", "yes" or "on", in any case).
func NoSandboxFromEnv(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(EnvNoSandbox))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// browserCandidates are Chromium-based executables searched in PATH, in order.
var browserCandidates = []string{
	"chrome-headless-shell",
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"brave-browser",
	"microsoft-edge",
}

// FindBrowser resolves the browser executable: the explicit path if given,
// then the CARDMAKER_CHROME environment variable, then well-known names in PATH.
func FindBrowser(explicit string) (string, error) {
	if explicit != "" {
		return checkExecutable(explicit)
	}
	if fromEnv := os.Getenv(EnvBrowserPath); fromEnv != "" {
		return checkExecutable(fromEnv)
	}

	for _, name := range browserCandidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no Chrome/Chromium browser found in PATH; install one, pass -chrome <path> or set %s",
		EnvBrowserPath)
}

func checkExecutable(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("browser %q: %w", path, err)
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("browser " + path + " is not an executable file")
	}

	return path, nil
}
