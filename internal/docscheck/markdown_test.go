package docscheck

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var (
	fencedCode = regexp.MustCompile("(?s)```.*?```")
	inlineCode = regexp.MustCompile("`[^`\n]*`")
	linkTarget = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	headingRow = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t]*$`)
	mdLinkText = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	slugDrop   = regexp.MustCompile(`[^\p{L}\p{N}\p{M} _-]`)
)

// stripCode removes fenced and inline code, so that text inside code samples
// is never mistaken for a link.
func stripCode(md string) string {
	return inlineCode.ReplaceAllString(fencedCode.ReplaceAllString(md, ""), "")
}

// links returns the target of every inline link and image of a Markdown text.
func links(md string) []string {
	var targets []string
	for _, m := range linkTarget.FindAllStringSubmatch(stripCode(md), -1) {
		targets = append(targets, m[1])
	}

	return targets
}

// slug converts a heading into the anchor GitHub generates for it: lower case,
// punctuation removed, spaces turned into hyphens. Letters with accents stay.
func slug(heading string) string {
	text := mdLinkText.ReplaceAllString(heading, "$1")
	text = strings.ReplaceAll(text, "`", "")
	text = slugDrop.ReplaceAllString(strings.ToLower(text), "")

	return strings.ReplaceAll(text, " ", "-")
}

// anchors returns every heading anchor of a Markdown text. Repeated headings
// get the -1, -2... suffixes GitHub adds.
func anchors(md string) map[string]bool {
	seen := map[string]int{}
	found := map[string]bool{}

	for _, m := range headingRow.FindAllStringSubmatch(fencedCode.ReplaceAllString(md, ""), -1) {
		base := slug(m[1])
		name := base
		if n := seen[base]; n > 0 {
			name = base + "-" + string(rune('0'+n))
		}
		seen[base]++
		found[name] = true
	}

	return found
}

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"Docker":                         "docker",
		"3. Design de referência":        "3-design-de-referência",
		"`POST /v1/render`":              "post-v1render",
		"O que é? Sim!":                  "o-que-é-sim",
		"A  B":                           "a--b",
		"Variantes de layout":            "variantes-de-layout",
		"[Guia](x.md) de temas":          "guia-de-temas",
		"Segurança e `-no-sandbox`":      "segurança-e--no-sandbox",
		"Entrada 1: cartas (cards.json)": "entrada-1-cartas-cardsjson",
	}

	for heading, want := range tests {
		if got := slug(heading); got != want {
			t.Errorf("slug(%q) = %q, want %q", heading, got, want)
		}
	}
}

func TestAnchors(t *testing.T) {
	md := "# Title\n\n## Same\n\ntext\n\n## Same\n\n```\n# not a heading\n```\n\n### Other `code`\n"

	got := anchors(md)

	for _, want := range []string{"title", "same", "same-1", "other-code"} {
		if !got[want] {
			t.Errorf("missing anchor %q in %v", want, got)
		}
	}
	if got["not-a-heading"] {
		t.Error("a heading inside a code block was counted")
	}
}

func TestLinksIgnoreCode(t *testing.T) {
	md := "See [guide](docs/A.md#x) and ![img](images/a.png \"title\").\n\n" +
		"```\n[not](a-link.md)\n```\n\nInline `[nor](this.md)` and [ok](B.md).\n"

	want := []string{"docs/A.md#x", "images/a.png", "B.md"}

	if got := links(md); !reflect.DeepEqual(got, want) {
		t.Errorf("links = %v, want %v", got, want)
	}
}
