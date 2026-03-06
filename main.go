package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/zalimeni/sp2md/cmd/sp2md"
)

func main() {
	if err := sp2md.Execute(); err != nil {
		var exitErr *sp2md.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.Error())
			os.Exit(exitErr.Code())
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
