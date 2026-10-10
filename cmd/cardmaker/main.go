// Command cardmaker is the CLI for Card Maker Go.
package main

import (
	"fmt"
	"os"

	"github.com/Bryant-Anjos/card-maker-go/pkg/cardmaker"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(cardmaker.Version)
		return
	}

	fmt.Fprintln(os.Stderr, "usage: cardmaker version")
	os.Exit(2)
}
