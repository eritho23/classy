package main

import (
	"context"
	"log"
	"os"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func main() {
	databaseUrl, exists := os.LookupEnv("DATABASE_URL")
	if !exists {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()

	db, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close(ctx)

	q := queries.New(db)

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Fatalf("could not begin tx")
	}

	qTx := q.WithTx(tx)

	defer tx.Commit(ctx)

	tx.Exec(ctx, "delete from session; delete from person; delete from grp;")
	grp, _ := qTx.CreateGroup(ctx, pgtype.Text{String: "230S", Valid: true})
	hash, _ := hashing.GenerateNewHash([]byte("a"))
	qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "erre",
	})
	qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "ström",
	})
	qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "ian",
	})
}
