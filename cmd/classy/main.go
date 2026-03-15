package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"classy/internal/generated/database"
	"classy/internal/handlers"
	"classy/internal/hashing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	// BEGIN SEED BLOCK //
	db.Exec(ctx, "delete from person; delete from grp;")
	grp, _ := q.CreateGroup(ctx, pgtype.Text{String: "230S", Valid: true})
	hash, _ := hashing.GenerateNewHash([]byte("erre"))
	q.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.ID,
		PasswordHash: hash,
		Username:     "erre",
	})
	// END SEED BLOCK //

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
	defer listener.Close()

	mux := http.NewServeMux()
	app.RegisterRouteHandlers(mux)

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}
