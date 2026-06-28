package main

import (
	"log/slog"
	"os"

	"github.com/kestfor/CrackHash/cmd/manager/app"
)

func main() {
	err := app.New().Execute()
	if err != nil {
		slog.Error("failed to start manager", "error", err)
		os.Exit(1)
	}
}
