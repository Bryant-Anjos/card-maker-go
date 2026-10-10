package docscheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// SourceHash fingerprints every file below the given directories of root: the
// relative path and the content of each one. The documentation images record
// the hash of the sources they were rendered from, so a test can tell when the
// theme changed and the images need regenerating.
func SourceHash(root string, dirs []string) (string, error) {
	var files []string

	for _, dir := range dirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))

			return nil
		})
		if err != nil {
			return "", fmt.Errorf("hash %s: %w", dir, err)
		}
	}

	sort.Strings(files)

	h := sha256.New()
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		content := sha256.Sum256(data)
		fmt.Fprintf(h, "%s\x00%x\n", rel, content)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ImageSources lists the directories (relative to the repository root) whose
// content the documentation images depend on: the theme, the sample data and
// the script that composes the images.
var ImageSources = []string{
	"themes/default",
	"testdata/showcase",
	"internal/devtools/docimages",
}
