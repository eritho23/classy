package handlers

import (
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *ClassyApplication) GetGroupGroupIdPersonPersonIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	suggestions, err := app.queries.GetSuggestionsByRegardingUserInGroup(r.Context(), queries.GetSuggestionsByRegardingUserInGroupParams{
		Caster: authStatus.PersonId,
		RegardingUid: pgtype.UUID{
			Bytes: targetPersonUUID,
			Valid: true,
		},
		GroupUid: groupRow.Uid,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = layouts.SuggestionsForPersonPage(authStatus, queries.Person{
		Uid:      targetPersonRow.Uid,
		Username: targetPersonRow.Username,
		Grp:      targetPersonRow.Grp,
	}, suggestions, csrf.Token(r)).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render suggestions for person page: %v", err)
	}
}
