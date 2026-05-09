// Package main demonstrates running apps in single-app and multi-app modes.
//
// Single-app mode (dedicated deployment):
//
//	MODE=single DATABASE_URL=postgres://... go run main.go
//
// Multi-app mode (shared deployment):
//
//	MODE=multi DATABASE_URL=postgres://... go run main.go
//
// Then test with:
//
//	# Single-app mode:
//	curl http://localhost:8080/api/info
//
//	# Multi-app mode:
//	curl -H "X-App-ID: example" http://localhost:8080/api/info
package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/grokify/systemforge/multiapp"
	"github.com/grokify/systemforge/multiapp/example"
)

func main() {
	// Get configuration from environment
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "single"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://localhost:5432/systemforge?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL") // Optional

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Determine server mode
	var serverMode multiapp.ServerMode
	switch mode {
	case "multi":
		serverMode = multiapp.MultiAppMode
	default:
		serverMode = multiapp.SingleAppMode
	}

	// Create server
	server, err := multiapp.NewServer(multiapp.Config{
		Mode:        serverMode,
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
		Logger:      logger,
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Register apps
	// In multi-app mode, you'd register multiple apps here
	// In single-app mode, you typically register just one

	exampleApp := example.NewExampleApp()
	if err := server.RegisterApp(exampleApp); err != nil {
		log.Fatalf("Failed to register example app: %v", err)
	}

	// In multi-app mode, you might register more apps:
	// if serverMode == multiapp.MultiAppMode {
	//     server.RegisterApp(app1.NewApp())
	//     server.RegisterApp(app2.NewApp())
	//     server.RegisterApp(app3.NewApp())
	// }

	logger.Info("starting server",
		"mode", serverMode,
		"addr", addr,
		"apps", server.Apps(),
	)

	// Run server (blocks until shutdown)
	if err := server.Run(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
