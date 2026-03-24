package handlers

import (
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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
