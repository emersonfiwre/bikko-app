package main

import (
	"bikko-app/internal/application"
	"log/slog"
	"os"
)

func main() {
	if err := application.Run("Bikkerfy", "8081"); err != nil {
		slog.Error("API stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
