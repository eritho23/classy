package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()

	databaseUrl, exists := os.LookupEnv("DATABASE_URL")
	if !exists {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	socketPath, exists := os.LookupEnv("HTTP_SOCKET_PATH")
	if !exists {
		fmt.Println("HTTP_SOCKET_PATH not set")
		os.Exit(1)
	}

	db, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		fmt.Println("HTTP_SOCKET_PATH not set")
		os.Exit(1)
	}
	defer db.Close(ctx)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to create listener: %v.\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
		w.WriteHeader(http.StatusOK)
	}))

	if err := http.Serve(listener, mux); err != nil {
		panic(err)
	}
}
