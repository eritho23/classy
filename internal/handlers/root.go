package handlers

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
	"classy/internal/layouts"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	sessionValue, err := hashing.GenerateSalt(32)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionValueHex := hex.EncodeToString(sessionValue)
	sessionValueHexHashHex := hashing.HashSessionValue(sessionValueHex)

	newSession, err := app.queries.CreateSession(r.Context(), queries.CreateSessionParams{
		Value:     sessionValueHexHashHex,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
		Person:    personRow.ID,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Expires:  newSession.ExpiresAt.Time,
		HttpOnly: true,
		Name:     "sessionid",
		Secure:   true,
		Value:    sessionValueHex,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
