// Package main is the entry point for the remote-ui server.
// This is a thin wrapper around the cli package.
// Spec: deployment.md
package main

import (
	"os"

	"github.com/zot/ui-engine/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
