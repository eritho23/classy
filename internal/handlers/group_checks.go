package handlers

import (
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func uuidEqual(a, b pgtype.UUID) bool {
	return a.Valid == b.Valid && a.Bytes == b.Bytes
}

func parsePathUUID(w http.ResponseWriter, r *http.Request, pathParam string) (uuid.UUID, bool) {
	pathValue := r.PathValue(pathParam)
	if pathValue == "" {
		w.WriteHeader(http.StatusBadRequest)
		return uuid.UUID{}, false
	}

	parsedUUID, err := uuid.Parse(pathValue)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return uuid.UUID{}, false
	}

	return parsedUUID, true
}

func (app *ClassyApplication) requireGroupMembership(
	w http.ResponseWriter,
	r *http.Request,
	authStatus middleware.AuthenticationStatus,
) (queries.GetGroupAndPersonPartOfGroupByGroupUidRow, bool) {
	groupUUID, ok := parsePathUUID(w, r, "groupId")
	if !ok {
		return queries.GetGroupAndPersonPartOfGroupByGroupUidRow{}, false
	}

	groupRow, err := app.queries.GetGroupAndPersonPartOfGroupByGroupUid(r.Context(), queries.GetGroupAndPersonPartOfGroupByGroupUidParams{
		PersonUid: authStatus.PersonId,
		GroupUid: pgtype.UUID{
			Bytes: groupUUID,
			Valid: true,
		},
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return queries.GetGroupAndPersonPartOfGroupByGroupUidRow{}, false
	}

	if !groupRow.PersonPartOfGroup {
		w.WriteHeader(http.StatusNotFound)
		return queries.GetGroupAndPersonPartOfGroupByGroupUidRow{}, false
	}

	return groupRow, true
}

func (app *ClassyApplication) requireTargetPersonInGroup(
	w http.ResponseWriter,
	r *http.Request,
	groupUID pgtype.UUID,
) (queries.GetPersonByUidRow, bool) {
	targetPersonUUID, ok := parsePathUUID(w, r, "personId")
	if !ok {
		return queries.GetPersonByUidRow{}, false
	}

	targetPersonRow, err := app.queries.GetPersonByUid(r.Context(), pgtype.UUID{
		Bytes: targetPersonUUID,
		Valid: true,
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return queries.GetPersonByUidRow{}, false
	}

	if !uuidEqual(targetPersonRow.Grp, groupUID) {
		w.WriteHeader(http.StatusNotFound)
		return queries.GetPersonByUidRow{}, false
	}

	return targetPersonRow, true
}
