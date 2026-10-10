package renderers_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/render/chrome"
	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/renderers"
)

func TestOpenFake(t *testing.T) {
	r, closeFn, err := renderers.Open(renderers.Fake, renderers.Options{})

	if err != nil || r == nil {
		t.Fatalf("Open(fake) = (%v, %v)", r, err)
	}
	if err := closeFn(); err != nil {
		t.Errorf("close: %v", err)
	}
}

func TestOpenUnknown(t *testing.T) {
	_, _, err := renderers.Open("magic", renderers.Options{})

	if err == nil || !strings.Contains(err.Error(), "magic") || !strings.Contains(err.Error(), "chrome") {
		t.Errorf("error = %v, want it to name the unknown renderer and the valid ones", err)
	}
}

func TestOpenChromeWithMissingBrowser(t *testing.T) {
	_, _, err := renderers.Open(renderers.Chrome, renderers.Options{BrowserPath: "/nope/chrome"})

	if err == nil || !strings.Contains(err.Error(), "/nope/chrome") {
		t.Errorf("error = %v", err)
	}
}

func TestOpenChrome(t *testing.T) {
	if _, err := chrome.FindBrowser(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}

	r, closeFn, err := renderers.Open(renderers.Chrome, renderers.Options{NoSandbox: chrome.NoSandboxFromEnv(os.Getenv)})
	if err != nil {
		t.Fatalf("Open(chrome): %v", err)
	}
	defer func() { _ = closeFn() }()

	if !strings.HasPrefix(r.Version(), "chrome-") {
		t.Errorf("Version = %q", r.Version())
	}
}
