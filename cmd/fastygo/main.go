package main

import (
	"fmt"
	"os"

	"github.com/fastygo/cli/internal/fasty"
)

func main() {
	if err := fasty.Main(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
