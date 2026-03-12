package handlers

import (
	queries "classy/internal/generated/database"
	"classy/internal/middleware"

	"github.com/jackc/pgx/v5"
)

type ClassyApplication struct {
	queries *queries.Queries
	db      *pgx.Conn
}

func NewClassyApplication(queries *queries.Queries, db *pgx.Conn) ClassyApplication {
	return ClassyApplication{
		queries: queries,
		db:      db,
	}
}
