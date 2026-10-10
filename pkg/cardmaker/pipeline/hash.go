package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"
)

// version is mixed into every cache key; bump it when the pipeline changes how
// it produces files, so stale caches are never trusted.
const version = "1"

// hashFS fingerprints every file of a file system (paths and contents).
func hashFS(fsys fs.FS) (string, error) {
	h := sha256.New()

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		file, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		content := sha256.New()
		if _, err := io.Copy(content, file); err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%x\n", path, content.Sum(nil))

		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// keyOf combines parts into one cache key. Each part is length-prefixed so
// different splits of the same characters never collide.
func keyOf(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(strconv.Itoa(len(part))))
		h.Write([]byte{':'})
		h.Write([]byte(part))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(v)
	}

	return strings.Join(parts, ",")
}
