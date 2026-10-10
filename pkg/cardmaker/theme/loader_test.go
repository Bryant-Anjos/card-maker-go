package theme_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

const themeJSON = `{
  "schemaVersion": 1,
  "name": "basic",
  "size": {"width": 750, "height": 1050},
  "template": "card.html.tmpl",
  "stylesheet": "style.css",
  "back": "back.html.tmpl",
  "rarities": [{"id": "common", "color": "#8b9099", "frame": "frames/common.svg"}],
  "elements": [{"id": "fire", "icon": "icons/fire.svg"}],
  "variants": [{"id": "fullart", "template": "fullart.html.tmpl"}]
}`

func completeTheme() fstest.MapFS {
	return fstest.MapFS{
		"theme.json":         {Data: []byte(themeJSON)},
		"card.html.tmpl":     {Data: []byte("<html></html>")},
		"fullart.html.tmpl":  {Data: []byte("<html></html>")},
		"style.css":          {Data: []byte("body{}")},
		"back.html.tmpl":     {Data: []byte("<html></html>")},
		"frames/common.svg":  {Data: []byte("<svg/>")},
		"icons/fire.svg":     {Data: []byte("<svg/>")},
		"fonts/unused.woff2": {Data: []byte("x")},
	}
}

func TestLoad(t *testing.T) {
	t.Run("complete theme", func(t *testing.T) {
		loaded, err := theme.Load(completeTheme())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaded.Name != "basic" {
			t.Errorf("name = %q", loaded.Name)
		}
		if _, err := fs.Stat(loaded.FS, "card.html.tmpl"); err != nil {
			t.Errorf("loaded FS cannot reach theme files: %v", err)
		}
	})

	t.Run("missing theme.json", func(t *testing.T) {
		fsys := completeTheme()
		delete(fsys, "theme.json")

		_, err := theme.Load(fsys)

		assertErrContains(t, err, "theme.json")
	})

	t.Run("invalid theme.json content", func(t *testing.T) {
		fsys := completeTheme()
		fsys["theme.json"] = &fstest.MapFile{Data: []byte(`{"schemaVersion":1,"bogus":true}`)}

		_, err := theme.Load(fsys)

		assertErrContains(t, err, "bogus")
	})

	missing := []struct {
		name string
		file string
		want string
	}{
		{"template", "card.html.tmpl", "template"},
		{"stylesheet", "style.css", "stylesheet"},
		{"back", "back.html.tmpl", "back"},
		{"rarity frame", "frames/common.svg", "rarities[0].frame"},
		{"element icon", "icons/fire.svg", "elements[0].icon"},
		{"variant template", "fullart.html.tmpl", "variants[0].template"},
	}
	for _, tt := range missing {
		t.Run("missing "+tt.name, func(t *testing.T) {
			fsys := completeTheme()
			delete(fsys, tt.file)

			_, err := theme.Load(fsys)

			assertErrContains(t, err, tt.want)
			assertErrContains(t, err, tt.file)
		})
	}

	t.Run("reports all missing files at once", func(t *testing.T) {
		fsys := completeTheme()
		delete(fsys, "card.html.tmpl")
		delete(fsys, "style.css")

		_, err := theme.Load(fsys)

		assertErrContains(t, err, "template")
		assertErrContains(t, err, "stylesheet")
	})

	t.Run("directory in place of file", func(t *testing.T) {
		fsys := completeTheme()
		delete(fsys, "style.css")
		fsys["style.css/inner.txt"] = &fstest.MapFile{Data: []byte("x")}

		_, err := theme.Load(fsys)

		assertErrContains(t, err, "stylesheet")
	})
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	for name, file := range completeTheme() {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, file.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := theme.LoadDir(dir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Name != "basic" {
		t.Errorf("name = %q", loaded.Name)
	}

	if _, err := theme.LoadDir(filepath.Join(dir, "nope")); err == nil {
		t.Error("expected error for missing directory")
	}
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}
