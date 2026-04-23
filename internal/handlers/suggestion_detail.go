package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	regardingUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return
	}

	regardingRow, ok := app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	suggestionUUID, ok := parsePathUUID(w, r, "suggestionId")
	if !ok {
		return
	}

	suggestionRow, err := app.queries.GetSuggestionByUidInGroupRegarding(r.Context(), queries.GetSuggestionByUidInGroupRegardingParams{
		SuggestionUid: pgtype.UUID{Bytes: suggestionUUID, Valid: true},
		RegardingUid:  pgtype.UUID{Bytes: regardingUUID, Valid: true},
		GroupUid:      groupRow.Uid,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	suggesterRow, err := app.queries.GetPersonByUid(r.Context(), suggestionRow.Suggester)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = layouts.SuggestionDetailPage(authStatus, regardingRow, suggesterRow, suggestionRow, csrf.Token(r)).Render(r.Context(), w)
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

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	regardingUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return
	}

	_, ok = app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	suggestionUUID, ok := parsePathUUID(w, r, "suggestionId")
	if !ok {
		return
	}

	suggestionRow, err := app.queries.GetSuggestionByUidInGroupRegarding(r.Context(), queries.GetSuggestionByUidInGroupRegardingParams{
		SuggestionUid: pgtype.UUID{Bytes: suggestionUUID, Valid: true},
		RegardingUid:  pgtype.UUID{Bytes: regardingUUID, Valid: true},
		GroupUid:      groupRow.Uid,
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

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	regardingUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return
	}

	regardingRow, ok := app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	suggestionUUID, ok := parsePathUUID(w, r, "suggestionId")
	if !ok {
		return
	}

	suggestionRow, err := app.queries.GetSuggestionByUidInGroupRegarding(r.Context(), queries.GetSuggestionByUidInGroupRegardingParams{
		SuggestionUid: pgtype.UUID{Bytes: suggestionUUID, Valid: true},
		RegardingUid:  pgtype.UUID{Bytes: regardingUUID, Valid: true},
		GroupUid:      groupRow.Uid,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
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

	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupRow.Uid, regardingRow.Uid), http.StatusSeeOther)
}

func (app *ClassyApplication) PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteVoteIdRemoveHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
		return
	}

	regardingUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return
	}

	regardingRow, ok := app.requireTargetPersonInGroup(w, r, groupRow.Uid)
	if !ok {
		return
	}

	suggestionUUID, ok := parsePathUUID(w, r, "suggestionId")
	if !ok {
		return
	}

	voteUUID, ok := parsePathUUID(w, r, "voteId")
	if !ok {
		return
	}

	vote, err := app.queries.GetVoteByUidInGroupRegardingSuggestion(r.Context(), queries.GetVoteByUidInGroupRegardingSuggestionParams{
		VoteUid:       pgtype.UUID{Bytes: voteUUID, Valid: true},
		SuggestionUid: pgtype.UUID{Bytes: suggestionUUID, Valid: true},
		RegardingUid:  pgtype.UUID{Bytes: regardingUUID, Valid: true},
		GroupUid:      groupRow.Uid,
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

	http.Redirect(w, r, fmt.Sprintf("/group/%s/person/%s", groupRow.Uid, regardingRow.Uid), http.StatusSeeOther)
}
