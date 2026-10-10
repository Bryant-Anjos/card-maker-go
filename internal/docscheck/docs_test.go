package docscheck

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const repoRoot = "../.."

func read(t *testing.T, rel string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}

	return string(data)
}

func exists(rel string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel)))

	return err == nil
}

// walk lists the files below dir (relative to the repository root) for which
// keep returns true, as slash-separated paths.
func walk(t *testing.T, dir string, keep func(rel string) bool) []string {
	t.Helper()

	var out []string
	err := filepath.WalkDir(filepath.Join(repoRoot, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, p)
		if rel = filepath.ToSlash(rel); keep(rel) {
			out = append(out, rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(out)

	return out
}

func isMarkdown(rel string) bool { return strings.HasSuffix(rel, ".md") }

func markdownFiles(t *testing.T) []string {
	t.Helper()

	files := walk(t, "docs", isMarkdown)
	for _, top := range []string{"README.md", "CLAUDE.md"} {
		if exists(top) {
			files = append(files, top)
		}
	}

	return files
}

// missing returns the wanted items that do not appear in text.
func missing(text string, wanted []string, format func(string) string) []string {
	var out []string
	for _, w := range wanted {
		if !strings.Contains(text, format(w)) {
			out = append(out, w)
		}
	}

	return out
}

func report(t *testing.T, what, doc, fix string, items []string) {
	t.Helper()

	if len(items) > 0 {
		t.Errorf("%s not documented in %s: %s\n\tfix: %s", what, doc, strings.Join(items, ", "), fix)
	}
}

func TestMarkdownLinksResolve(t *testing.T) {
	for _, file := range markdownFiles(t) {
		md := read(t, file)
		own := anchors(md)

		for _, target := range links(md) {
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			pathPart, anchor, _ := strings.Cut(target, "#")

			if pathPart == "" { // link to a heading of the same file
				if !own[anchor] {
					t.Errorf("%s: anchor #%s does not match any heading", file, anchor)
				}
				continue
			}

			resolved := path.Join(path.Dir(file), pathPart)
			if !exists(resolved) {
				t.Errorf("%s: link to %q is broken (no file %s)", file, target, resolved)
				continue
			}
			if anchor != "" && isMarkdown(resolved) && !anchors(read(t, resolved))[anchor] {
				t.Errorf("%s: link %q points to a heading that does not exist in %s", file, target, resolved)
			}
		}
	}
}

func TestDocumentationIsIndexed(t *testing.T) {
	index := read(t, "docs/README.md")

	var unlisted []string
	for _, doc := range walk(t, "docs", func(rel string) bool { return isMarkdown(rel) && path.Dir(rel) == "docs" }) {
		name := path.Base(doc)
		if name != "README.md" && !strings.Contains(index, "("+name) {
			unlisted = append(unlisted, name)
		}
	}
	report(t, "documents", "docs/README.md", "add each one to the index table", unlisted)

	if !strings.Contains(read(t, "README.md"), "(docs/README.md") {
		t.Error("README.md must link to the documentation index docs/README.md")
	}
}

func TestEveryDocImageIsUsed(t *testing.T) {
	var corpus strings.Builder
	for _, file := range markdownFiles(t) {
		corpus.WriteString(read(t, file))
	}

	var orphans []string
	for _, img := range walk(t, "docs/images", func(rel string) bool { return !strings.HasSuffix(rel, "SOURCE_HASH") }) {
		if !strings.Contains(corpus.String(), path.Base(img)) {
			orphans = append(orphans, img)
		}
	}
	report(t, "images", "any document", "reference them in a document or delete them", orphans)
}

func TestCLIIsDocumented(t *testing.T) {
	doc := read(t, "docs/CLI.md")

	var flags []string
	for _, src := range []string{"cmd/cardmaker/cli.go", "cmd/cardmaker-server/server.go"} {
		flags = append(flags, flagsIn(read(t, src))...)
	}
	report(t, "flags", "docs/CLI.md", "add a row to the flag tables, as `-name`",
		missing(doc, unique(flags), func(f string) string { return "`-" + f })) // matches `-name` and `-name <value>`

	report(t, "subcommands", "docs/CLI.md", "add a section for `cardmaker <name>`",
		missing(doc, submatches(commandCase, read(t, "cmd/cardmaker/cli.go"), 1), func(c string) string { return "cardmaker " + c }))

	var envs []string
	for _, root := range []string{"cmd", "pkg"} {
		for _, file := range walk(t, root, func(rel string) bool { return strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go") }) {
			envs = append(envs, submatches(envName, read(t, file), 1)...)
		}
	}
	report(t, "environment variables", "docs/CLI.md", "add them to the environment variables table",
		missing(doc, unique(envs), func(e string) string { return "`" + e + "`" }))
}

func TestAPIRoutesAreDocumented(t *testing.T) {
	var routes []string
	for _, m := range routeCall.FindAllStringSubmatch(read(t, "pkg/cardmaker/server/server.go"), -1) {
		routes = append(routes, m[1]+" "+m[2])
	}

	report(t, "API routes", "docs/API.md", "add a section for each route, titled with `METHOD /path`",
		missing(read(t, "docs/API.md"), unique(routes), func(r string) string { return r }))
}

func TestPackagesAreDocumented(t *testing.T) {
	isSource := func(rel string) bool { return strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go") }

	dirs := map[string]bool{"schema": true, "themes/default": true}
	for _, root := range []string{"cmd", "pkg", "internal"} {
		for _, file := range walk(t, root, isSource) {
			dirs[path.Dir(file)] = true
		}
	}

	var list []string
	for d := range dirs {
		list = append(list, d)
	}
	sort.Strings(list)

	report(t, "packages and directories", "docs/ARQUITETURA.md", "add a row to the package map",
		missing(read(t, "docs/ARQUITETURA.md"), list, func(d string) string { return "`" + d + "`" }))
}

func TestSchemasAreDocumented(t *testing.T) {
	var names []string
	for _, f := range walk(t, "schema", func(rel string) bool { return strings.HasSuffix(rel, ".schema.json") }) {
		names = append(names, path.Base(f))
	}

	report(t, "JSON Schemas", "docs/CONTRATO.md", "mention each schema file",
		missing(read(t, "docs/CONTRATO.md"), names, func(n string) string { return n }))
}

func TestMakeTargetsAreDocumented(t *testing.T) {
	targets := submatches(makeTarget, read(t, "Makefile"), 1)

	report(t, "make targets", "docs/DESENVOLVIMENTO.md", "add each one to the make targets table, as `make name`",
		missing(read(t, "docs/DESENVOLVIMENTO.md"), unique(targets), func(n string) string { return "`make " + n + "`" }))
}

func TestDocImagesMatchTheTheme(t *testing.T) {
	want, err := SourceHash(repoRoot, ImageSources)
	if err != nil {
		t.Fatalf("SourceHash: %v", err)
	}
	got := strings.TrimSpace(read(t, "docs/images/SOURCE_HASH"))

	if got != want {
		t.Errorf("the documentation images are out of date: they were rendered from sources with hash %s but the theme or sample data now hash to %s\n\tfix: run `make docs-images` and commit docs/images/", got, want)
	}
}

func TestAssistantInstructionsPointToTheDocumentationRules(t *testing.T) {
	claude := read(t, "CLAUDE.md")

	for _, want := range []string{"docs/README.md", "make check"} {
		if !strings.Contains(claude, want) {
			t.Errorf("CLAUDE.md must mention %q", want)
		}
	}
}
