package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
)

type AccountLabel string

func RequireAuth(queries *queries.Queries, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionid, err := r.Cookie("sessionid")
		if err != nil {
			// gPodder seems to use stateless client.
			username, password, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			accountWithHashField, err := queries.GetPersonPasswordHashByUsername(r.Context(), username)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			correct, err := hashing.CheckPassword(accountWithHashField.PasswordHash, []byte(password))
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !correct {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			newCtx := context.WithValue(r.Context(), AccountLabel("account"), username)
			newReq := r.WithContext(newCtx)

			next.ServeHTTP(w, newReq)
		} else {
			sessionValueHexHashHex := hashing.HashSessionValue(sessionid.Value)
			session, err := queries.GetSessionByValue(r.Context(), sessionValueHexHashHex)
			if err != nil {
				log.Printf("denied account %s due to non-existent session", session.Person)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if time.Now().After(session.ExpiresAt.Time) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			log.Printf("proceeding with user: %s", session.Person)
			newCtx := context.WithValue(r.Context(), AccountLabel("account"), session.ID)
			newReq := r.WithContext(newCtx)

			next.ServeHTTP(w, newReq)
		}
	})
}
