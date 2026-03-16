package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"
)

type AuthenticationStatus struct {
	IsAuthenticated bool
	PersonName      string
}
type AuthenticationStatusKey string

func CheckAuth(queries *queries.Queries, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionid, err := r.Cookie("sessionid")
		if err != nil {
			// We set the context authenticated to false.
			newCtx := context.WithValue(r.Context(), AuthenticationStatusKey("authentication_status"), AuthenticationStatus{
				IsAuthenticated: false,
			})
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

			newCtx := context.WithValue(r.Context(), AuthenticationStatusKey("authentication_status"), AuthenticationStatus{
				IsAuthenticated: true,
				PersonName:      session.Username,
			})

			newReq := r.WithContext(newCtx)

			next.ServeHTTP(w, newReq)
		}
	})
}
