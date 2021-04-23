package main

import (
	"fmt"
	"os"

	"github.com/kaleido-io/firefly/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
