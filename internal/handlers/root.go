package handlers

import (
	"encoding/hex"
	"errors"
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
	router.HandleFunc("GET /group/{groupId}/person/{personId}/suggest", app.GetGroupGroupIdPersonPersonIdSuggestHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggest", app.PostGroupGroupIdPersonPersonIdSuggestHandler)
	router.HandleFunc("GET /group/{groupId}/person/{personId}", app.GetGroupGroupIdPersonPersonIdHandler)
	router.HandleFunc("GET /group/{groupId}/person/{personId}/suggestion/{suggestionId}", app.GetGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}/vote", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}/vote/{voteId}/remove", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteVoteIdRemoveHandler)

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

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdSuggestHandler(w http.ResponseWriter, r *http.Request) {
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

	if personRow.Uid.Bytes == authStatus.PersonId.Bytes {
		http.Redirect(w, r, fmt.Sprintf("/group/%s", groupId), http.StatusSeeOther)
		return
	}

	err = layouts.SuggestionSubmitPage(authStatus, queries.Person{
		Uid:      personRow.Uid,
		Username: personRow.Username,
		Grp:      personRow.Grp,
	}).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render suggestion page: %v", err)
	}
}

func (app *ClassyApplication) PostGroupGroupIdPersonPersonIdSuggestHandler(w http.ResponseWriter, r *http.Request) {
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

	if personRow.Uid.Bytes == authStatus.PersonId.Bytes {
		http.Redirect(w, r, fmt.Sprintf("/group/%s", groupId), http.StatusSeeOther)
		return
	}

	suggestionExists, err := app.queries.ExistsSuggestionOnTargetByUserById(r.Context(), queries.ExistsSuggestionOnTargetByUserByIdParams{
		SuggesterUid: authStatus.PersonId,
		RegardingUid: pgtype.UUID{
			Valid: true,
			Bytes: targetPersonUuid,
		},
		GroupUid: groupRow.Uid,
	})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if suggestionExists {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("You have already made a suggestion for this person..."))
		return
	}

	suggestionValue := r.FormValue("value")
	if suggestionValue == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestion, err := app.queries.CreateSuggestion(r.Context(), queries.CreateSuggestionParams{
		Suggester: authStatus.PersonId,
		Regarding: pgtype.UUID{
			Valid: true,
			Bytes: targetPersonUuid,
		},
		Suggestion: pgtype.Text{
			String: suggestionValue,
			Valid:  true,
		},
		Motivation: pgtype.Text{
			String: r.FormValue("motivation"),
			Valid:  true,
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupId, suggestion.Regarding), http.StatusSeeOther)
}

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
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

	targetPersonRow, err := app.queries.GetPersonByUid(r.Context(), pgtype.UUID{
		Bytes: targetPersonUuid,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if targetPersonRow.Uid.Bytes == authStatus.PersonId.Bytes {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestions, err := app.queries.GetSuggestionsByRegardingUser(r.Context(), queries.GetSuggestionsByRegardingUserParams{
		Caster: authStatus.PersonId,
		Regarding: pgtype.UUID{
			Bytes: targetPersonUuid,
			Valid: true,
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = layouts.SuggestionsForPersonPage(authStatus, queries.Person{
		Uid:      targetPersonRow.Uid,
		Username: targetPersonRow.Username,
		Grp:      targetPersonRow.Grp,
	}, suggestions).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render suggestions for person page: %v", err)
	}
}

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
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

	targetPersonRow, err := app.queries.GetPersonByUid(r.Context(), pgtype.UUID{
		Bytes: targetPersonUuid,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if targetPersonRow.Uid.Bytes == authStatus.PersonId.Bytes {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestionId := r.PathValue("suggestionId")
	if suggestionId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestionUuid, err := uuid.Parse(suggestionId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestion, err := app.queries.GetSuggestionByUid(r.Context(), pgtype.UUID{
		Bytes: suggestionUuid,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = layouts.SuggestionDetailPage(authStatus, queries.Person{
		Uid:      targetPersonRow.Uid,
		Username: targetPersonRow.Username,
	}, queries.Person{
		Uid:      authStatus.PersonId,
		Username: authStatus.PersonName,
	}, suggestion).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render suggestion detail page: %v", err)
	}
}

func (app *ClassyApplication) PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestionId := r.PathValue("suggestionId")
	if suggestionId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestionUuid, err := uuid.Parse(suggestionId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestionRow, err := app.queries.GetSuggestionByUid(r.Context(), pgtype.UUID{
		Bytes: suggestionUuid,
		Valid: true,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if authStatus.PersonId != suggestionRow.Suggester {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestion := r.FormValue("suggestion")
	if suggestion == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	motivation := r.FormValue("motivation")

	err = app.queries.UpdateSuggestion(r.Context(), queries.UpdateSuggestionParams{
		Suggestion: pgtype.Text{
			String: suggestion,
			Valid:  true,
		},
		Motivation: pgtype.Text{
			String: motivation,
			Valid:  true,
		},
		Uid: suggestionRow.Uid,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
}

func (app *ClassyApplication) PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestionId := r.PathValue("suggestionId")
	if suggestionId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestionUuid, err := uuid.Parse(suggestionId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	suggestionRow, err := app.queries.GetSuggestionByUid(r.Context(), pgtype.UUID{
		Bytes: suggestionUuid,
		Valid: true,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	groupId, err := app.queries.GetGroupByPersonUid(r.Context(), authStatus.PersonId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if authStatus.PersonId == suggestionRow.Suggester || authStatus.PersonId == suggestionRow.Regarding {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("You cannot create a vote now."))
		return
	}

	_, err = app.queries.CreateVote(r.Context(), queries.CreateVoteParams{
		Caster:           authStatus.PersonId,
		TargetSuggestion: suggestionRow.Uid,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupId, suggestionRow.Regarding), http.StatusSeeOther)
}

func (app *ClassyApplication) PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteVoteIdRemoveHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupId, err := app.queries.GetGroupByPersonUid(r.Context(), authStatus.PersonId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	voteId := r.PathValue("voteId")
	if voteId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	voteUuid, err := uuid.Parse(voteId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vote, err := app.queries.GetVoteByUid(r.Context(), pgtype.UUID{
		Bytes: voteUuid,
		Valid: true,
	})

	if vote.Caster != authStatus.PersonId {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = app.queries.DeleteVoteByUid(r.Context(), vote.Uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupId, vote.Regarding), http.StatusSeeOther)
}
