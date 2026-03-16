package handlers

import (
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
	"classy/internal/layouts"
	"classy/internal/middleware"

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
	router.HandleFunc("GET /logout", app.GetLogoutHandler)

	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
		if !authStatus.IsAuthenticated {
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprintf(w, "OK for user: %s", authStatus.PersonName)
			if err != nil {
				slog.Warn("Error in writing health string???")
			}
		}
	})
}

func (app *ClassyApplication) GetRootHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		layouts.RootPage(authStatus, nil).Render(r.Context(), w)
		return
	}

	grp, err := app.queries.GetGroupByUsername(r.Context(), authStatus.PersonName)
	if err != nil {
		log.Printf("failed to get group for user: %s", authStatus.PersonName)
		layouts.RootPage(authStatus, nil).Render(r.Context(), w)
		return
	}

	layouts.RootPage(authStatus, &queries.Grp{
		ID:   grp.GroupID,
		Name: grp.GroupName,
	}).Render(r.Context(), w)
}

func (app *ClassyApplication) GetLoginHandler(w http.ResponseWriter, r *http.Request) {
	layouts.LoginPage("", middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
}

const incorrectCredentialsMsg = "invalid credentials"

func (app *ClassyApplication) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	providedUsername := r.FormValue("username")
	providedPassword := r.FormValue("password")
	if providedUsername == "" || providedPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		layouts.LoginPage("both username and password must be provided", middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
		return
	}

	personRow, err := app.queries.GetPersonPasswordHashByUsername(r.Context(), providedUsername)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		layouts.LoginPage(incorrectCredentialsMsg, middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
		return
	}

	match, err := hashing.CheckPassword(personRow.PasswordHash, []byte(providedPassword))
	if err != nil || !match {
		w.WriteHeader(http.StatusUnauthorized)
		layouts.LoginPage(incorrectCredentialsMsg, middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
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
		Path:     "/",
		Secure:   true,
		Value:    sessionValueHex,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *ClassyApplication) GetLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("sessionid")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sessionValueHexHashHex := hashing.HashSessionValue(session.Value)
	oldSession, err := app.queries.GetSessionByValue(r.Context(), sessionValueHexHashHex)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if time.Now().After(oldSession.ExpiresAt.Time) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = app.queries.DeleteSessionById(r.Context(), oldSession.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
		Name:     "sessionid",
		Path:     "/",
		Secure:   true,
		Value:    "",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
