package main

import (
	"os"

	"buzz-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
