package main

import (
	"log"

	"github.com/Coku2015/veeamgo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
