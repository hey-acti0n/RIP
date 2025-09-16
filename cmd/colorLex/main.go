package main

import (
	"RIP/internal/api"
	"log"
)

func main() {
	if err := api.Run(); err != nil {
		log.Fatal(err)
	}
}
