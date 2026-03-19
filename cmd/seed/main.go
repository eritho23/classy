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

	q := queries.New(db)

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Fatalf("could not begin tx")
	}

	qTx := q.WithTx(tx)

	_, err = tx.Exec(ctx, "delete from session; delete from person; delete from grp;")
	if err != nil {
		log.Printf("could not do initial cleaning of db before seed: %v", err)
	}

	grp, err := qTx.CreateGroup(ctx, pgtype.Text{String: "230S", Valid: true})
	if err != nil {
		log.Printf("could not create seeded group: %v", err)
	}

	hash, err := hashing.GenerateNewHash([]byte("a"))
	if err != nil {
		log.Printf("could not generate hash: %v", err)
	}

	_, err1 := qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "erre",
	})
	_, err2 := qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "ström",
	})
	_, err3 := qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "ian",
	})
	_, err4 := qTx.CreatePerson(ctx, queries.CreatePersonParams{
		Grp:          grp.Uid,
		PasswordHash: hash,
		Username:     "smul",
	})

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		log.Printf("could not create all three people: %v, %v, %v, %v", err1, err2, err3, err4)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("seeding transaction failed: %v", err)
	}

	if err := db.Close(ctx); err != nil {
		log.Printf("failed to close db: %v", err)
	}
}
