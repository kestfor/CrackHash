package main

import (
	"log/slog"
	"os"

	"github.com/kestfor/CrackHash/cmd/worker/app"
)

func main() {
	err := app.New().Execute()
	if err != nil {
		slog.Error("failed to start worker", "error", err)
		os.Exit(1)
	}
}
