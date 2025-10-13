package main

import (
	"log"

	"github.com/veeamgo/veeamgo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
