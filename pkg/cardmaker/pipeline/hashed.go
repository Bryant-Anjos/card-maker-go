package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker/model"
)

const (
	hashLength = 8
	// filesPrefix marks the cache entries that remember the file names of a
	// hashed build, since they depend on the content and cannot be recomputed.
	filesPrefix = "files:"
)

// hashedPath inserts the content hash before the extension:
// "cards/a@2x.webp" becomes "cards/a@2x.3fa9c1d2.webp".
func hashedPath(p, format string, data []byte) string {
	sum := sha256.Sum256(data)

	return strings.TrimSuffix(p, "."+format) + "." + hex.EncodeToString(sum[:])[:hashLength] + "." + format
}

// recordedFiles is the cache value of a hashed output: the key it was built
// with and the file name of each planned image, in planned order.
type recordedFiles struct {
	Key   string   `json:"key"`
	Paths []string `json:"paths"`
}

// recordFiles remembers the final names of a hashed output.
func (b *build) recordFiles(cacheName, key string, images []model.Image) {
	if !b.opts.HashedNames {
		return
	}
	paths := make([]string, len(images))
	for i, img := range images {
		paths[i] = img.Path
	}
	data, err := json.Marshal(recordedFiles{Key: key, Paths: paths})
	if err != nil {
		return // cannot happen for strings; worst case the output is rebuilt
	}
	b.cache.set(filesPrefix+cacheName, string(data))
}

// recordedImages returns the planned images renamed to the recorded hashed
// names, or false when nothing usable was recorded for this key.
func (b *build) recordedImages(cacheName, key string, planned []model.Image) ([]model.Image, bool) {
	raw, ok := b.cache.get(filesPrefix + cacheName)
	if !ok {
		return nil, false
	}
	var rec recordedFiles
	if err := json.Unmarshal([]byte(raw), &rec); err != nil || rec.Key != key || len(rec.Paths) != len(planned) {
		return nil, false
	}

	images := make([]model.Image, len(planned))
	for i, img := range planned {
		img.Path = rec.Paths[i]
		images[i] = img
	}

	return images, true
}

// removeOrphans deletes the image files of older generations from the locale
// directory: the PNG and WebP files of cards/ and the back images at the root
// that the new manifest does not list. It looks at regular files directly
// inside those two directories only, so the manifest, the cache and anything
// else stay untouched.
func removeOrphans(localeDir string, manifest model.Manifest) error {
	keep := map[string]bool{}
	for _, img := range manifest.Back {
		keep[img.Path] = true
	}
	for _, card := range manifest.Cards {
		for _, img := range card.Images {
			keep[img.Path] = true
		}
	}

	root, err := orphansIn(localeDir, "", backBaseName, keep)
	if err != nil {
		return err
	}
	cards, err := orphansIn(localeDir, cardsDir, "", keep)
	if err != nil {
		return err
	}

	var errs []error
	for _, name := range append(root, cards...) {
		if err := os.Remove(filepath.Join(localeDir, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("remove old images: %w", err)
	}

	return nil
}

// orphansIn lists the unlisted image files of one directory, as manifest-style
// paths. When namePrefix is set, only files starting with it are considered.
func orphansIn(localeDir, dir, namePrefix string, keep map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(localeDir, dir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, e := range entries {
		name := e.Name()
		rel := imagePath(dir, name)
		if !e.Type().IsRegular() || !isImageFile(name) || !strings.HasPrefix(name, namePrefix) || keep[rel] {
			continue
		}
		orphans = append(orphans, rel)
	}

	return orphans, nil
}

func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))

	return ext == ".png" || ext == ".webp"
}
