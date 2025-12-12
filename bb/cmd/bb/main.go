package main

import (
	"os"

	"github.com/cli/bb/v2/internal/ghcmd"
)

func main() {
	code := ghcmd.Main()
	os.Exit(int(code))
}
