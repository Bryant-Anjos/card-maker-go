package docscheck

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceHash(t *testing.T) {
	dirs := []string{"themes/default", "testdata/showcase"}
	base := map[string]string{
		"themes/default/style.css":          "body{}",
		"themes/default/fonts/a.woff2":      "font",
		"testdata/showcase/cards.json":      "{}",
		"testdata/showcase/art/lunaris.png": "png",
		"unrelated/file.txt":                "ignored",
	}

	root := t.TempDir()
	writeTree(t, root, base)
	first, err := SourceHash(root, dirs)
	if err != nil {
		t.Fatalf("SourceHash: %v", err)
	}

	t.Run("is deterministic", func(t *testing.T) {
		again, err := SourceHash(root, dirs)
		if err != nil || again != first {
			t.Errorf("hash changed between identical calls: %q vs %q (%v)", first, again, err)
		}
	})

	t.Run("same content elsewhere gives the same hash", func(t *testing.T) {
		other := t.TempDir()
		writeTree(t, other, base)

		got, err := SourceHash(other, dirs)
		if err != nil || got != first {
			t.Errorf("hash = %q, want %q (%v)", got, first, err)
		}
	})

	t.Run("a changed file changes the hash", func(t *testing.T) {
		other := t.TempDir()
		changed := map[string]string{}
		for k, v := range base {
			changed[k] = v
		}
		changed["themes/default/style.css"] = "body{margin:1px}"
		writeTree(t, other, changed)

		got, _ := SourceHash(other, dirs)
		if got == first {
			t.Error("hash did not change when a theme file changed")
		}
	})

	t.Run("an added file changes the hash", func(t *testing.T) {
		other := t.TempDir()
		added := map[string]string{"themes/default/new.svg": "<svg/>"}
		for k, v := range base {
			added[k] = v
		}
		writeTree(t, other, added)

		got, _ := SourceHash(other, dirs)
		if got == first {
			t.Error("hash did not change when a file was added")
		}
	})

	t.Run("a renamed file changes the hash", func(t *testing.T) {
		other := t.TempDir()
		renamed := map[string]string{}
		for k, v := range base {
			renamed[k] = v
		}
		delete(renamed, "themes/default/style.css")
		renamed["themes/default/other.css"] = "body{}"
		writeTree(t, other, renamed)

		got, _ := SourceHash(other, dirs)
		if got == first {
			t.Error("hash did not change when a file was renamed")
		}
	})

	t.Run("files outside the listed directories are ignored", func(t *testing.T) {
		other := t.TempDir()
		extra := map[string]string{"unrelated/file.txt": "different content", "docs/x.md": "x"}
		for k, v := range base {
			if _, ok := extra[k]; !ok {
				extra[k] = v
			}
		}
		writeTree(t, other, extra)

		got, _ := SourceHash(other, dirs)
		if got != first {
			t.Error("hash changed because of a file outside the listed directories")
		}
	})

	t.Run("a missing directory is an error", func(t *testing.T) {
		if _, err := SourceHash(t.TempDir(), dirs); err == nil {
			t.Error("expected an error for a missing directory")
		}
	})
}
