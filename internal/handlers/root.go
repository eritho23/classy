package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
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
	router.HandleFunc("GET /login", app.GetLoginHandler)
	router.HandleFunc("POST /login", app.PostLoginHandler)

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

func (app *ClassyApplication) GetLoginHandler(w http.ResponseWriter, r *http.Request) {
	layouts.LoginPage("").Render(r.Context(), w)
}

const incorrectCredentialsMsg = "invalid credentials"

func (app *ClassyApplication) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	providedUsername := r.FormValue("username")
	providedPassword := r.FormValue("password")
	if providedUsername == "" || providedPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		layouts.LoginPage("both username and password must be provided").Render(r.Context(), w)
		return
	}

	personRow, err := app.queries.GetPersonPasswordHashByUsername(r.Context(), providedUsername)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		layouts.LoginPage(incorrectCredentialsMsg).Render(r.Context(), w)
		return
	}

	match, err := hashing.CheckPassword(personRow.PasswordHash, []byte(providedPassword))
	if err != nil || !match {
		w.WriteHeader(http.StatusUnauthorized)
		layouts.LoginPage(incorrectCredentialsMsg).Render(r.Context(), w)
		return
	}
}
