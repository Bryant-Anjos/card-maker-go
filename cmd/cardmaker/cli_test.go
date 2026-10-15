package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
)

const (
	cardsFile = "../../testdata/cards/cards.json"
	themeDir  = "../../testdata/themes/basic"
)

func execute(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)

	return code, out.String(), errOut.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := execute("version")

	if code != 0 || strings.TrimSpace(out) == "" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"unknown command", []string{"frobnicate"}},
		{"unknown flag", []string{"build", "-nope"}},
		{"bad scales", []string{"build", "-cards", cardsFile, "-theme", themeDir, "-out", "x", "-scales", "a,b", "-renderer", "fake"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := execute(tt.args...)

			if code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
			if strings.TrimSpace(errOut) == "" {
				t.Error("expected a message on stderr")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		code, out, errOut := execute("validate", "-cards", cardsFile, "-theme", themeDir)

		if code != 0 {
			t.Fatalf("code = %d, stderr = %s", code, errOut)
		}
		if !strings.Contains(out, "5 cards") {
			t.Errorf("stdout = %q, want it to mention 5 cards", out)
		}
	})

	t.Run("invalid inputs", func(t *testing.T) {
		code, _, errOut := execute("validate", "-cards", cardsFile, "-theme", "does-not-exist")

		if code != 1 || !strings.Contains(errOut, "does-not-exist") {
			t.Errorf("code = %d, stderr = %q", code, errOut)
		}
	})

	t.Run("missing required flags", func(t *testing.T) {
		code, _, errOut := execute("validate")

		if code != 1 || !strings.Contains(errOut, "required") {
			t.Errorf("code = %d, stderr = %q", code, errOut)
		}
	})
}

func TestBuildWithFakeRenderer(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")

	code, stdout, stderr := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", out,
		"-renderer", "fake", "-scales", "1,2", "-formats", "png")

	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, rel := range []string{"pt-BR/manifest.json", "pt-BR/cards/vulpin.png", "pt-BR/cards/vulpin@2x.png", "pt-BR/back.png"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing output %s: %v", rel, err)
		}
	}
	if !strings.Contains(stdout, "rendered 12") {
		t.Errorf("stdout = %q, want a summary with 'rendered 12'", stdout)
	}

	code, stdout, _ = execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", out,
		"-renderer", "fake", "-scales", "1,2", "-formats", "png")
	if code != 0 || !strings.Contains(stdout, "rendered 0") || !strings.Contains(stdout, "skipped 12") {
		t.Errorf("second run: code = %d, stdout = %q, want everything skipped", code, stdout)
	}
}

func TestBuildHashedNames(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	args := []string{"build", "-cards", cardsFile, "-theme", themeDir, "-out", out,
		"-renderer", "fake", "-scales", "1", "-formats", "png", "-hashed-names"}

	code, stdout, stderr := execute(args...)

	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(out, "pt-BR", "cards", "vulpin.png")); err == nil {
		t.Error("an unhashed name was written")
	}
	manifest, err := os.ReadFile(filepath.Join(out, "pt-BR", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`"path": "cards/vulpin\.[0-9a-f]{8}\.png"`).Match(manifest) {
		t.Errorf("manifest does not list the hashed name:\n%s", manifest)
	}

	code, stdout, _ = execute(args...)
	if code != 0 || !strings.Contains(stdout, "rendered 0") {
		t.Errorf("second run: code = %d, stdout = %q", code, stdout)
	}
}

func TestBuildChromeRendererWithMissingBrowser(t *testing.T) {
	code, _, errOut := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", t.TempDir(),
		"-chrome", "/nope/chrome")

	if code != 1 || !strings.Contains(errOut, "/nope/chrome") {
		t.Errorf("code = %d, stderr = %q, want the missing browser path in the error", code, errOut)
	}
}

func TestBuildWithChromeRenderer(t *testing.T) {
	if _, err := chrome.FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}
	out := t.TempDir()

	code, stdout, stderr := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", out, "-scales", "1")

	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "rendered 6") {
		t.Errorf("stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(out, "pt-BR", "cards", "drakarion.png")); err != nil {
		t.Errorf("missing output: %v", err)
	}
}

func TestBuildAcceptsNoSandboxFlag(t *testing.T) {
	code, _, stderr := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", t.TempDir(),
		"-renderer", "fake", "-scales", "1", "-no-sandbox")

	if code != 0 {
		t.Errorf("code = %d, stderr = %s", code, stderr)
	}
}

func TestUnknownRenderer(t *testing.T) {
	code, _, errOut := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", t.TempDir(), "-renderer", "magic")

	if code != 1 || !strings.Contains(errOut, "magic") {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
}

func TestBuildWebPQualityFlag(t *testing.T) {
	out := t.TempDir()
	base := []string{"build", "-cards", cardsFile, "-theme", themeDir, "-out", out,
		"-renderer", "fake", "-scales", "1", "-formats", "webp"}

	if code, _, stderr := execute(append(base, "-webp-quality", "60")...); code != 0 {
		t.Fatalf("quality 60: code = %d, stderr = %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(out, "pt-BR", "cards", "vulpin.webp")); err != nil {
		t.Errorf("missing webp output: %v", err)
	}

	code, _, stderr := execute(append(base, "-webp-quality", "101")...)
	if code != 1 || !strings.Contains(stderr, "quality") {
		t.Errorf("quality 101: code = %d, stderr = %q, want a quality error", code, stderr)
	}
}

func TestParseCommaList(t *testing.T) {
	if _, err := parseScales("1,x"); err == nil {
		t.Error("expected error for non-numeric scale")
	}
	if got, err := parseScales(" 1 , 2 "); err != nil || len(got) != 2 {
		t.Errorf("parseScales = (%v, %v)", got, err)
	}
	if got := splitList("png, webp,"); len(got) != 2 || got[1] != "webp" {
		t.Errorf("splitList = %v", got)
	}
}
