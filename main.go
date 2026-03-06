package main

import (
	"os"

	"github.com/zalimeni/sp2md/cmd/sp2md"
)

func main() {
	if err := sp2md.Execute(); err != nil {
		os.Exit(1)
	}
}
