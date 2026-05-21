package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *ClassyApplication) GetScoreboardHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		clearSessionAndRedirectToLogin(w, r)
		return
	}

	tx, err := app.db.BeginTx(r.Context(), pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		log.Println("failed to acquire database tx context in scoreboard handler")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tx.Rollback(r.Context())
	}()

	qTx := app.queries.WithTx(tx)

	points, err := qTx.GetTotalPoints(r.Context())
	if err != nil {
		log.Println("failed to get points")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	challenges, err := qTx.GetAllChallengesWithCompleter(r.Context())
	if err != nil {
		log.Println("failed to get challenges")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	totalPointsInt := int(points)

	_ = tx.Commit(r.Context())

	err = layouts.PointTable(authStatus, totalPointsInt, challenges, csrf.Token(r)).Render(r.Context(), w)
	if err != nil {
		log.Println("failed to render point table page")
	}
}

func (app *ClassyApplication) GetScoreboardChallengeCompleteHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		clearSessionAndRedirectToLogin(w, r)
		return
	}

	challengeIdStr := r.PathValue("challengeId")
	var challengeId pgtype.UUID
	err := challengeId.Scan(challengeIdStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	challenges, err := app.queries.GetAllChallenges(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var challenge *queries.Challenge
	for _, c := range challenges {
		if c.Uid == challengeId {
			challenge = &c
			break
		}
	}

	if challenge == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = layouts.ChallengeCompletePage(authStatus, *challenge, csrf.Token(r)).Render(r.Context(), w)
	if err != nil {
		log.Println("failed to render challenge complete page")
	}
}

func (app *ClassyApplication) PostScoreboardChallengeCompleteHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		clearSessionAndRedirectToLogin(w, r)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	challengeIdStr := r.PathValue("challengeId")
	var challengeId pgtype.UUID
	err = challengeId.Scan(challengeIdStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	challenges, err := app.queries.GetAllChallenges(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var challenge *queries.Challenge
	for _, c := range challenges {
		if c.Uid == challengeId {
			challenge = &c
			break
		}
	}
	if challenge == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if challenge.CompletedBy.Valid {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	extraPointsStr := r.FormValue("extra_points")
	extraPoints := int32(0)
	if challenge.ExtraPointsAvailable && extraPointsStr != "" {
		ep, err := strconv.ParseInt(extraPointsStr, 10, 32)
		if err == nil && ep > 0 {
			extraPoints = int32(ep)
		}
	}

	err = app.queries.CompleteChallengeWithExtraPoints(r.Context(), queries.CompleteChallengeWithExtraPointsParams{
		PersonUid:    authStatus.PersonId,
		ChallengeUid: challengeId,
		ExtraPoints:  extraPoints,
	})
	if err != nil {
		log.Println("failed to complete challenge")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashPart := fmt.Sprintf("#b%vc%v", challenge.BatchNumber, challenge.AssignedNumber)
	http.Redirect(w, r, "/scoreboard"+hashPart, http.StatusSeeOther)
}
