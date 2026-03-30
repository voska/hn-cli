package main

import (
	"fmt"
	"os"

	"github.com/voska/hn-cli/internal/cmd"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(cmd.ExitCode(err))
	}
}
