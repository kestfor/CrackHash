package main

import (
	"log"

	"github.com/kestfor/CrackHash/cmd/manager/app"
)

func main() {
	err := app.New().Execute()
	if err != nil {
		log.Fatal(err)
	}
}
