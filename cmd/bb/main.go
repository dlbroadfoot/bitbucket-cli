package main

import (
	"os"

	"github.com/dlbroadfoot/bitbucket-cli/internal/ghcmd"
)

func main() {
	code := ghcmd.Main()
	os.Exit(int(code))
}
