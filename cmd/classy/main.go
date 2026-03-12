package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"classy/internal/generated/database"
	"classy/internal/handlers"

	"github.com/jackc/pgx/v5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	databaseUrl, exists := os.LookupEnv("DATABASE_URL")
	if !exists {
		log.Fatal("DATABASE_URL not set")
	}

	socketPath, exists := os.LookupEnv("HTTP_SOCKET_PATH")
	if !exists {
		log.Fatal("HTTP_SOCKET_PATH not set")
	}

	db, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close(ctx)

	q := queries.New(db)
	app := handlers.NewClassyApplication(q, db)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to create listener: %v.\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	app.RegisterRouteHandlers(mux)

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}
