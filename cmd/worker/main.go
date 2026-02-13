package main

import (
	"log"

	"github.com/kestfor/CrackHash/cmd/worker/app"
)

func main() {
	err := app.New().Execute()
	if err != nil {
		log.Fatal(err)
	}
}
