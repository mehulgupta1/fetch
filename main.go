package main

import (
	"os"

	"github.com/mehulgupta1/fetch/internal/cli"
)

func main() {
	piped := false
	if fi, err := os.Stdin.Stat(); err == nil {
		piped = (fi.Mode() & os.ModeCharDevice) == 0
	}
	os.Exit(cli.Run(os.Args[1:], cli.Deps{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
		StdinPiped: piped,
	}))
}
