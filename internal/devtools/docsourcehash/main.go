// Command docsourcehash prints the fingerprint of the sources the documentation
// images are rendered from. `make docs-images` stores it in
// docs/images/SOURCE_HASH.
package main

import (
	"fmt"
	"os"

	"github.com/Bryant-Anjos/card-maker-go/internal/docscheck"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	hash, err := docscheck.SourceHash(root, docscheck.ImageSources)
	if err != nil {
		fmt.Fprintln(os.Stderr, "docsourcehash:", err)
		os.Exit(1)
	}

	fmt.Println(hash)
}
