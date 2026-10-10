// Command cardmaker is the CLI for Card Maker Go.
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
