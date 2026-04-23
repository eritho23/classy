package handlers

import (
	"encoding/hex"
	"log"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	invalidLoginMsg        = "Ogiltiga inloggningsuppgifter."
	maxFormBodyBytes int64 = 1 << 20
)

func (app *ClassyApplication) GetLoginHandler(w http.ResponseWriter, r *http.Request) {
	err := layouts.LoginPage("", middleware.GetAuthenticationStatusFromRequestContext(r), csrf.Token(r)).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render login page template: %v", err)
	}
}

func (app *ClassyApplication) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)

	providedUsername := r.FormValue("username")
	providedPassword := r.FormValue("password")
	if providedUsername == "" || providedPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		err := layouts.LoginPage("Både användarnamn och lösenord måste anges.", middleware.GetAuthenticationStatusFromRequestContext(r), csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render login page template: %v", err)
		}
		return
	}

	personRow, err := app.queries.GetPersonPasswordHashByUsername(r.Context(), providedUsername)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		err = layouts.LoginPage(invalidLoginMsg, middleware.GetAuthenticationStatusFromRequestContext(r), csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render login page template: %v", err)
		}
		return
	}

	match, err := hashing.CheckPassword(personRow.PasswordHash, []byte(providedPassword))
	if err != nil || !match {
		w.WriteHeader(http.StatusUnauthorized)
		err = layouts.LoginPage(invalidLoginMsg, middleware.GetAuthenticationStatusFromRequestContext(r), csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render login page template: %v", err)
		}
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
		Person:    personRow.Uid,
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
		SameSite: http.SameSiteStrictMode,
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

	err = app.queries.DeleteSessionByUid(r.Context(), oldSession.Uid)
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
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Value:    "",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
