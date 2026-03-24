package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	regardingId := r.PathValue("personId")
	if regardingId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	regardingUuid, err := uuid.Parse(regardingId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	regardingRow, err := app.queries.GetPersonByUid(r.Context(), pgtype.UUID{
		Bytes: regardingUuid,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
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
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	suggesterRow, err := app.queries.GetPersonByUid(r.Context(), suggestionRow.Suggester)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = layouts.SuggestionDetailPage(authStatus, regardingRow, suggesterRow, suggestionRow).Render(r.Context(), w)
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

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)

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

	newSuggestionValue := r.FormValue("suggestion")
	if newSuggestionValue == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newSuggestionMotivation := r.FormValue("motivation")

	err = app.queries.UpdateSuggestion(r.Context(), queries.UpdateSuggestionParams{
		Suggestion: pgtype.Text{
			String: newSuggestionValue,
			Valid:  true,
		},
		Motivation: pgtype.Text{
			String: newSuggestionMotivation,
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

	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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
