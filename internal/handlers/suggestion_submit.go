package handlers

import (
	"fmt"
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)

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
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
