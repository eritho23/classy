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
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
