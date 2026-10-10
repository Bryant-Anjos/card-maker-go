package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestBuildDefaultRendererNotAvailableYet(t *testing.T) {
	code, _, errOut := execute("build", "-cards", cardsFile, "-theme", themeDir, "-out", t.TempDir())

	if code != 1 || !strings.Contains(errOut, "-renderer fake") {
		t.Errorf("code = %d, stderr = %q, want a hint to use -renderer fake", code, errOut)
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
