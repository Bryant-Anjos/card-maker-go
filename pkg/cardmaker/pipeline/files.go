package pipeline

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// imageName returns the file name of an image: "id.png" at 1x and "id@2x.png"
// for larger scales.
func imageName(base string, scale int, format string) string {
	if scale == 1 {
		return fmt.Sprintf("%s.%s", base, format)
	}

	return fmt.Sprintf("%s@%dx.%s", base, scale, format)
}

// imagePath joins a manifest directory and a file name with forward slashes,
// as the manifest contract requires.
func imagePath(dir, name string) string {
	if dir == "" {
		return name
	}

	return path.Join(dir, name)
}

// writeFileAtomic writes data to a temporary file and renames it into place,
// so readers never observe a half-written file.
func writeFileAtomic(target string, data []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)

		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)

		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)

		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)

		return err
	}

	return nil
}

func allExist(root string, paths []string) bool {
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			return false
		}
	}

	return true
}
