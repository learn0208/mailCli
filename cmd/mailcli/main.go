package main

import (
	"os"

	"github.com/learn0208/mailcli/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		os.Exit(1)
	}
}
