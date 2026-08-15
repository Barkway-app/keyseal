package main

import (
	"fmt"
	"os"

	"github.com/jrpbuilds/keyseal/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if exitCoder, ok := err.(interface{ ExitCode() int }); ok {
			if err.Error() != "" {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(exitCoder.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
