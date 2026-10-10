package chrome_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
)

// recordingBrowser is a stand-in executable that records its arguments and
// exits with an error. It lets the tests see exactly which flags the renderer
// launches the browser with, without needing a real browser.
func recordingBrowser(t *testing.T) (exe, argsFile string) {
	t.Helper()

	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args.txt")
	exe = filepath.Join(dir, "fake-browser")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CARDMAKER_TEST_ARGS_FILE\"\nexit 1\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARDMAKER_TEST_ARGS_FILE", argsFile)

	return exe, argsFile
}

func launchedArgs(t *testing.T, opts chrome.Options, argsFile string) []string {
	t.Helper()

	// The fake browser exits immediately, so New is expected to fail; only the
	// arguments it was launched with matter here.
	if _, err := chrome.New(opts); err == nil {
		t.Fatal("expected New to fail with a browser that exits immediately")
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("the browser was never launched: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func contains(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}

	return false
}

func TestSandboxIsOnByDefault(t *testing.T) {
	exe, argsFile := recordingBrowser(t)

	args := launchedArgs(t, chrome.Options{BrowserPath: exe}, argsFile)

	if contains(args, "--no-sandbox") {
		t.Errorf("sandbox disabled by default: %v", args)
	}
	if !contains(args, "--headless") {
		t.Errorf("browser not launched headless: %v", args)
	}
}

func TestNoSandboxOption(t *testing.T) {
	exe, argsFile := recordingBrowser(t)

	args := launchedArgs(t, chrome.Options{BrowserPath: exe, NoSandbox: true}, argsFile)

	if !contains(args, "--no-sandbox") {
		t.Errorf("--no-sandbox missing from %v", args)
	}
}

func TestNoSandboxFromEnv(t *testing.T) {
	tests := map[string]bool{
		"1": true, "true": true, "TRUE": true, "yes": true, "on": true,
		"": false, "0": false, "false": false, "no": false, "maybe": false,
	}

	for value, want := range tests {
		got := chrome.NoSandboxFromEnv(func(key string) string {
			if key == chrome.EnvNoSandbox {
				return value
			}
			return ""
		})

		if got != want {
			t.Errorf("%s=%q -> %v, want %v", chrome.EnvNoSandbox, value, got, want)
		}
	}
}
