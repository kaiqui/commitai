package main

import (
	"os"

	"github.com/kaiqui/commitai/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
