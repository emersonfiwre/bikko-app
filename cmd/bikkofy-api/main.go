package main

import (
	"bikko-app/internal/application"
	"log/slog"
	"os"
)

func main() {
	if err := application.Run("Bikkofy", "8080"); err != nil {
		slog.Error("API stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
