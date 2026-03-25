package handlers

import (
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"
)

func (app *ClassyApplication) GetGroupGroupIdHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupRow, ok := app.requireGroupMembership(w, r, authStatus)
	if !ok {
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
