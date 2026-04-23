package middleware

import (
	"context"
	"net/http"
	"time"

	queries "classy/internal/generated/database"
	"classy/internal/hashing"

	"github.com/jackc/pgx/v5/pgtype"
)

type AuthenticationStatus struct {
	IsAuthenticated bool
	PersonName      string
	PersonId        pgtype.UUID
}
type authenticationStatusKeyType string

const authenticationStatusKey authenticationStatusKeyType = "authentication_status"

func GetAuthenticationStatusFromRequestContext(r *http.Request) AuthenticationStatus {
	val := r.Context().Value(authenticationStatusKey)
	if val == nil {
		return AuthenticationStatus{}
	}

	status, ok := val.(*AuthenticationStatus)
	if !ok || status == nil {
		return AuthenticationStatus{}
	}

	return *status
}

func CheckAuth(queries *queries.Queries, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionid, err := r.Cookie("sessionid")
		if err != nil {
			// We set the context authenticated to false.
			newCtx := context.WithValue(r.Context(), authenticationStatusKey, &AuthenticationStatus{
				IsAuthenticated: false,
			})
			newReq := r.WithContext(newCtx)

			next.ServeHTTP(w, newReq)
		} else {
			sessionValueHexHashHex := hashing.HashSessionValue(sessionid.Value)
			session, err := queries.GetSessionByValue(r.Context(), sessionValueHexHashHex)
			if err != nil || time.Now().After(session.ExpiresAt.Time) {
				newCtx := context.WithValue(r.Context(), authenticationStatusKey, &AuthenticationStatus{
					IsAuthenticated: false,
				})
				newReq := r.WithContext(newCtx)

				next.ServeHTTP(w, newReq)

				return
			}

			newCtx := context.WithValue(r.Context(), authenticationStatusKey, &AuthenticationStatus{
				IsAuthenticated: true,
				PersonName:      session.Username,
				PersonId:        session.Person,
			})

			newReq := r.WithContext(newCtx)

			next.ServeHTTP(w, newReq)
		}
	})
}
