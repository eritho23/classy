package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"classy/internal/credentials"
	"classy/internal/generated/database"
	"classy/internal/handlers"
	"classy/internal/middleware"

	"github.com/jackc/pgx/v5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	databaseUrl, err := credentials.ReadCredential("database_url")
	if err != nil {
		var ok bool
		databaseUrl, ok = os.LookupEnv("DATABASE_URL")
		if !ok {
			log.Fatal("failed to lookup credentials dir and DATABASE_URL is not set")
		}
	}

	socketPath, exists := os.LookupEnv("HTTP_SOCKET_PATH")
	if !exists {
		log.Fatal("HTTP_SOCKET_PATH not set")
	}

	origin, exists := os.LookupEnv("ORIGIN")
	if !exists {
		log.Fatal("ORIGIN not set")
	}

	db, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	q := queries.New(db)
	app := handlers.NewClassyApplication(q, db)

	_, err = os.Stat(socketPath)
	if err == nil {
		err = os.Remove(socketPath)
		if err != nil {
			log.Fatalf("failed to clear old socket: %v", err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to create listener: %v.\n", err)
	}

	mux := http.NewServeMux()
	app.RegisterRouteHandlers(mux)

	crossOriginProtection := http.NewCrossOriginProtection()
	err = crossOriginProtection.AddTrustedOrigin(origin)
	if err != nil {
		log.Fatalf("Failed to configure trusted origin: %v", err)
	}
	crossOriginProtection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("CSRF check failed"))
	}))

	muxWithMiddleware := middleware.CheckAuth(q, mux)
	server := &http.Server{
		Handler:           crossOriginProtection.Handler(muxWithMiddleware),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}

	if err := db.Close(ctx); err != nil {
		log.Printf("failed to close db: %v", err)
	}

	if err := listener.Close(); err != nil {
		log.Printf("failed to close listener: %v", err)
	}
}
