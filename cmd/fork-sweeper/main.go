package main

import (
	"github.com/rednafi/fork-sweeper/src"
	"os"
)

// Ldflags filled by goreleaser
var version = "dev"

func main() {
	cliConfig := src.NewCLIConfig(os.Stdout, os.Stderr, version)
	errCode := cliConfig.CLI(os.Args[1:])
	os.Exit(errCode)
}
