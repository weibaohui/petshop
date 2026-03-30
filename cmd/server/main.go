package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Project: petshop")

	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Application initialization
	return nil
}
