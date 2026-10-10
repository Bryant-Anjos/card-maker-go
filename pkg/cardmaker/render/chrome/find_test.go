package chrome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestFindBrowser(t *testing.T) {
	t.Run("explicit path wins", func(t *testing.T) {
		exe := fakeExecutable(t, t.TempDir(), "my-chrome")
		t.Setenv(EnvBrowserPath, "/should/not/be/used")

		got, err := FindBrowser(exe)

		if err != nil || got != exe {
			t.Errorf("FindBrowser = (%q, %v), want %q", got, err, exe)
		}
	})

	t.Run("explicit path must exist", func(t *testing.T) {
		_, err := FindBrowser("/nope/chrome")

		if err == nil || !strings.Contains(err.Error(), "/nope/chrome") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("environment variable", func(t *testing.T) {
		exe := fakeExecutable(t, t.TempDir(), "env-chrome")
		t.Setenv(EnvBrowserPath, exe)

		got, err := FindBrowser("")

		if err != nil || got != exe {
			t.Errorf("FindBrowser = (%q, %v), want %q", got, err, exe)
		}
	})

	t.Run("searches PATH", func(t *testing.T) {
		dir := t.TempDir()
		exe := fakeExecutable(t, dir, "chromium")
		t.Setenv(EnvBrowserPath, "")
		t.Setenv("PATH", dir)

		got, err := FindBrowser("")

		if err != nil || got != exe {
			t.Errorf("FindBrowser = (%q, %v), want %q", got, err, exe)
		}
	})

	t.Run("nothing found explains how to fix", func(t *testing.T) {
		t.Setenv(EnvBrowserPath, "")
		t.Setenv("PATH", t.TempDir())

		_, err := FindBrowser("")

		if err == nil || !strings.Contains(err.Error(), "-chrome") || !strings.Contains(err.Error(), EnvBrowserPath) {
			t.Errorf("error = %v, want a hint about -chrome and %s", err, EnvBrowserPath)
		}
	})
}
