package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	// "classy/internal/middleware"

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

func (app *ClassyApplication) RegisterRouteHandlers(router *http.ServeMux) {
	router.HandleFunc("GET /", app.GetRootHandler)

	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, "OK")
		if err != nil {
			slog.Warn("Error in writing health string???")
		}
	})
}

func (app *ClassyApplication) GetRootHandler(w http.ResponseWriter, r *http.Request) {
	layouts.RootPage().Render(r.Context(), w)
}
