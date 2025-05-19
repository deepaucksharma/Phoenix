// The validate_policy.go script validates policy.yaml files against the schema.
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/sa-omf/pkg/policy"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate_policy.go <path-to-policy.yaml>")
		os.Exit(1)
	}

	policyPath := os.Args[1]

	// Read the policy file
	data, err := os.ReadFile(policyPath)
	if err != nil {
		fmt.Printf("Error reading policy file: %v\n", err)
		os.Exit(1)
	}

	// Validate against schema
	err = policy.ValidatePolicy(data)
	if err != nil {
		fmt.Printf("Policy validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Policy file %s is valid\n", policyPath)
}
