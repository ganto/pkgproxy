package main

import (
	"os"

	"github.com/ganto/pkgproxy/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
