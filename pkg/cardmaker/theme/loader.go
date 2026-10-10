// Package theme loads a theme directory: its theme.json definition plus the
// template, stylesheet, frames and icons it references.
package theme

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

// DefinitionFile is the name of the theme definition inside a theme directory.
const DefinitionFile = "theme.json"

// Theme is a validated theme definition together with the file system that
// holds its assets, so renderers can read templates and images from it.
type Theme struct {
	model.Theme
	FS fs.FS
}

// LoadDir loads the theme stored in a directory on disk.
func LoadDir(dir string) (Theme, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return Theme{}, fmt.Errorf("theme directory: %w", err)
	}
	if !info.IsDir() {
		return Theme{}, fmt.Errorf("theme directory: %q is not a directory", dir)
	}

	return Load(os.DirFS(dir))
}

// Load reads theme.json from fsys, validates it and verifies that every file
// it references exists, reporting all missing files at once.
func Load(fsys fs.FS) (Theme, error) {
	file, err := fsys.Open(DefinitionFile)
	if err != nil {
		return Theme{}, fmt.Errorf("open %s: %w", DefinitionFile, err)
	}
	defer file.Close()

	definition, err := model.ParseTheme(file)
	if err != nil {
		return Theme{}, fmt.Errorf("%s: %w", DefinitionFile, err)
	}
	if err := checkReferencedFiles(fsys, definition); err != nil {
		return Theme{}, err
	}

	return Theme{Theme: definition, FS: fsys}, nil
}

type reference struct {
	field string
	path  string
}

func references(t model.Theme) []reference {
	refs := []reference{
		{"template", t.Template},
		{"stylesheet", t.Stylesheet},
	}
	if t.Back != "" {
		refs = append(refs, reference{"back", t.Back})
	}
	for i, r := range t.Rarities {
		refs = append(refs, reference{fmt.Sprintf("rarities[%d].frame", i), r.Frame})
	}
	for i, e := range t.Elements {
		refs = append(refs, reference{fmt.Sprintf("elements[%d].icon", i), e.Icon})
	}

	return refs
}

func checkReferencedFiles(fsys fs.FS, t model.Theme) error {
	var errs []error

	for _, ref := range references(t) {
		info, err := fs.Stat(fsys, ref.path)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%s: file %q not found in theme", ref.field, ref.path))
		case info.IsDir():
			errs = append(errs, fmt.Errorf("%s: %q is a directory, expected a file", ref.field, ref.path))
		}
	}

	return errors.Join(errs...)
}
