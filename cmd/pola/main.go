package main

import (
	"os"

	cli "github.com/polagonow/pola/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
