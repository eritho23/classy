package handlers

import (
	"net/http"
	"time"
)

func clearSessionAndRedirectToLogin(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
