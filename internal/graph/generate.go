// +build ignore

// This file is used to generate GraphQL code
// Run: go run generate.go

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	// Change to graph directory
	if err := os.Chdir("internal/graph"); err != nil {
		fmt.Printf("Error changing directory: %v\n", err)
		os.Exit(1)
	}

	// Run gqlgen generate
	cmd := exec.Command("go", "run", "github.com/99designs/gqlgen", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running gqlgen: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("GraphQL code generated successfully!")
}

