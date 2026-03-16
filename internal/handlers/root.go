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

	"github.com/google/uuid"
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
	router.HandleFunc("GET /group/{groupId}", app.GetGroupGroupIdHandler)
	router.HandleFunc("GET /group/{groupId}/suggest/{personId}", app.GetGroupGroupIdSuggestPersonIdHandler)

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
		err := layouts.RootPage(authStatus, nil).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render root page template: %v", err)
		}
		return
	}

	grp, err := app.queries.GetGroupByUsername(r.Context(), authStatus.PersonName)
	if err != nil {
		log.Printf("failed to get group for user: %s", authStatus.PersonName)
		err = layouts.RootPage(authStatus, nil).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render root page template: %v", err)
		}
		return
	}

	err = layouts.RootPage(authStatus, &queries.Grp{
		Uid:  grp.GroupUid,
		Name: grp.GroupName,
	}).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render root page template: %v", err)
	}
}

func (app *ClassyApplication) GetLoginHandler(w http.ResponseWriter, r *http.Request) {
	err := layouts.LoginPage("", middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render login page template: %v", err)
	}
}

const incorrectCredentialsMsg = "invalid credentials"

func (app *ClassyApplication) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	providedUsername := r.FormValue("username")
	providedPassword := r.FormValue("password")
	if providedUsername == "" || providedPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		err := layouts.LoginPage("both username and password must be provided", middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render login page template: %v", err)
		}
		return
	}

	personRow, err := app.queries.GetPersonPasswordHashByUsername(r.Context(), providedUsername)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		err = layouts.LoginPage(incorrectCredentialsMsg, middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render login page template: %v", err)
		}
		return
	}

	match, err := hashing.CheckPassword(personRow.PasswordHash, []byte(providedPassword))
	if err != nil || !match {
		w.WriteHeader(http.StatusUnauthorized)
		err = layouts.LoginPage(incorrectCredentialsMsg, middleware.GetAuthenticationStatusFromRequestContext(r)).Render(r.Context(), w)
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
		Secure:   true,
		Value:    "",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *ClassyApplication) GetGroupGroupIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupId := r.PathValue("groupId")
	if groupId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupUuid, err := uuid.Parse(groupId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupRow, err := app.queries.GetGroupAndPersonPartOfGroupByGroupUid(r.Context(), queries.GetGroupAndPersonPartOfGroupByGroupUidParams{
		PersonUid: authStatus.PersonId,
		GroupUid: pgtype.UUID{
			Bytes: groupUuid,
			Valid: true,
		},
	})

	if !groupRow.PersonPartOfGroup {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	students, err := app.queries.GetStudentsAndSuggestionCountsByGrp(r.Context(), groupRow.Uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = layouts.GroupPage(authStatus, queries.Grp{
		Uid:  groupRow.Uid,
		Name: groupRow.Name,
	}, students).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render group page template: %v", err)
	}
}

func (app *ClassyApplication) GetGroupGroupIdSuggestPersonIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupId := r.PathValue("groupId")
	if groupId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupUuid, err := uuid.Parse(groupId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupRow, err := app.queries.GetGroupAndPersonPartOfGroupByGroupUid(r.Context(), queries.GetGroupAndPersonPartOfGroupByGroupUidParams{
		PersonUid: authStatus.PersonId,
		GroupUid: pgtype.UUID{
			Bytes: groupUuid,
			Valid: true,
		},
	})

	if !groupRow.PersonPartOfGroup {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	targetPersonId := r.PathValue("personId")
	if targetPersonId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	targetPersonUuid, err := uuid.Parse(targetPersonId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	personRow, err := app.queries.GetPersonByUid(r.Context(), pgtype.UUID{
		Bytes: targetPersonUuid,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = layouts.SuggestionPage(authStatus, queries.Person{
		Uid:      personRow.Uid,
		Username: personRow.Username,
		Grp:      personRow.Grp,
	}).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render suggestion page: %v", err)
	}
}
