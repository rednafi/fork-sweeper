package main

import (
	"github.com/rednafi/fork-sweeper/src"
	"os"
	"text/tabwriter"
)

// Ldflags filled by goreleaser
var version = "dev"

func main() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	defer w.Flush()

	cliConfig := src.NewCLIConfig(w, version, os.Exit)

	cliConfig.CLI(os.Args[1:])
}
