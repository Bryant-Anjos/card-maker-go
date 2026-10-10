package theme_test

import (
	"encoding/xml"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/theme"
)

const defaultThemeDir = "../../../themes/default"

// The example elements of the default theme, in the order of theme.json.
var defaultElements = []string{"beast", "fire", "water", "nature", "wind", "light", "dark", "electric", "earth", "ice", "mystic"}

func loadDefaultTheme(t *testing.T) theme.Theme {
	t.Helper()

	th, err := theme.LoadDir(defaultThemeDir)
	if err != nil {
		t.Fatalf("load default theme: %v", err)
	}

	return th
}

func TestDefaultThemeElements(t *testing.T) {
	th := loadDefaultTheme(t)

	var ids []string
	for _, e := range th.Elements {
		ids = append(ids, e.ID)
		if label, _ := e.Label.Resolve("pt-BR", "pt-BR"); label == "" {
			t.Errorf("element %q has no pt-BR label", e.ID)
		}
	}
	if !slices.Equal(ids, defaultElements) {
		t.Errorf("elements = %v, want %v", ids, defaultElements)
	}
	nature, ok := th.Element("nature")
	if !ok {
		t.Fatal("element nature not found")
	}
	if label, _ := nature.Label.Resolve("pt-BR", "pt-BR"); label != "Natureza" {
		t.Errorf("nature label = %q, want Natureza", label)
	}
}

var svgID = regexp.MustCompile(`\bid="([^"]+)"`)

// Element icons are inlined into the card page next to the frame, so they
// must be well-formed 48x48 SVGs whose internal ids cannot clash with the
// frame's or with each other's: every id starts with icon-<element id>.
func TestDefaultThemeIconsAreWellFormed(t *testing.T) {
	th := loadDefaultTheme(t)

	for _, e := range th.Elements {
		t.Run(e.ID, func(t *testing.T) {
			data, err := fs.ReadFile(th.FS, e.Icon)
			if err != nil {
				t.Fatalf("read %s: %v", e.Icon, err)
			}

			var root struct {
				XMLName xml.Name
				ViewBox string `xml:"viewBox,attr"`
			}
			if err := xml.Unmarshal(data, &root); err != nil {
				t.Fatalf("%s is not well-formed XML: %v", e.Icon, err)
			}
			if root.XMLName.Local != "svg" || root.ViewBox != "0 0 48 48" {
				t.Errorf("%s: root = <%s viewBox=%q>, want <svg viewBox=%q>", e.Icon, root.XMLName.Local, root.ViewBox, "0 0 48 48")
			}
			if want := "icons/" + e.ID + ".svg"; e.Icon != want {
				t.Errorf("icon path = %q, want %q", e.Icon, want)
			}
			for _, m := range svgID.FindAllStringSubmatch(string(data), -1) {
				if !strings.HasPrefix(m[1], "icon-"+e.ID) {
					t.Errorf("%s: id %q must start with %q", e.Icon, m[1], "icon-"+e.ID)
				}
			}
		})
	}
}
