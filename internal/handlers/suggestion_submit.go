package handlers

import (
	"fmt"
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdSuggestHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	targetPersonRow, ok := app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	if uuidEqual(targetPersonRow.Uid, authStatus.PersonId) {
		groupId := r.PathValue("groupId")
		http.Redirect(w, r, fmt.Sprintf("/group/%s", groupId), http.StatusSeeOther)
		return
	}

	err := layouts.SuggestionSubmitPage(authStatus, queries.Person{
		Uid:      targetPersonRow.Uid,
		Username: targetPersonRow.Username,
		Grp:      targetPersonRow.Grp,
	}, csrf.Token(r)).Render(r.Context(), w)
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

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	targetPersonUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return
	}

	targetPersonRow, ok := app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	if uuidEqual(targetPersonRow.Uid, authStatus.PersonId) {
		groupId := r.PathValue("groupId")
		http.Redirect(w, r, fmt.Sprintf("/group/%s", groupId), http.StatusSeeOther)
		return
	}

	suggestionExists, err := app.queries.ExistsSuggestionOnTargetByUserById(r.Context(), queries.ExistsSuggestionOnTargetByUserByIdParams{
		SuggesterUid: authStatus.PersonId,
		RegardingUid: pgtype.UUID{
			Valid: true,
			Bytes: targetPersonUUID,
		},
		GroupUid: groupRow.Uid,
	})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if suggestionExists {
		w.WriteHeader(http.StatusUnauthorized)
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
			Bytes: targetPersonUUID,
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

	groupId := r.PathValue("groupId")
	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupId, suggestion.Regarding), http.StatusSeeOther)
}
