package main

import (
	"fmt"
	"os"

	"github.com/yourorg/sa-omf/pkg/policy"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s POLICY_FILE\n", os.Args[0])
		os.Exit(1)
	}
	if _, err := policy.LoadPolicy(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
