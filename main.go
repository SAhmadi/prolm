package main

import (
	"os"

	"github.com/prolm/prolm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
