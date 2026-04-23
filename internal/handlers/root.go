package handlers

import (
	"log"
	"net/http"

	queries "classy/internal/generated/database"
	"classy/internal/layouts"
	"classy/internal/middleware"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"
)

type ClassyApplication struct {
	queries *queries.Queries
	db      *pgx.Conn
}

func NewClassyApplication(queries *queries.Queries, db *pgx.Conn) ClassyApplication {
	return ClassyApplication{
		queries: queries,
		db:      db,
	}
}

func (app *ClassyApplication) RegisterRouteHandlers(router *http.ServeMux) {
	router.HandleFunc("GET /", app.GetRootHandler)
	router.HandleFunc("GET /login", app.GetLoginHandler)
	router.HandleFunc("POST /login", app.PostLoginHandler)
	router.HandleFunc("POST /logout", app.GetLogoutHandler)
	router.HandleFunc("GET /group/{groupId}", app.GetGroupGroupIdHandler)
	router.HandleFunc("GET /group/{groupId}/person/{personId}/suggest", app.GetGroupGroupIdPersonPersonIdSuggestHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggest", app.PostGroupGroupIdPersonPersonIdSuggestHandler)
	router.HandleFunc("GET /group/{groupId}/person/{personId}", app.GetGroupGroupIdPersonPersonIdHandler)
	router.HandleFunc("GET /group/{groupId}/person/{personId}/suggestion/{suggestionId}", app.GetGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}/vote", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteHandler)
	router.HandleFunc("POST /group/{groupId}/person/{personId}/suggestion/{suggestionId}/vote/{voteId}/remove", app.PostGroupGroupIdPersonPersonIdSuggestionSuggestionIdVoteVoteIdRemoveHandler)
}

func (app *ClassyApplication) GetRootHandler(w http.ResponseWriter, r *http.Request) {
	authStatus := middleware.GetAuthenticationStatusFromRequestContext(r)
	if !authStatus.IsAuthenticated {
		err := layouts.RootPage(authStatus, nil, csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render root page template: %v", err)
		}
		return
	}

	groupUID, err := app.queries.GetGroupByPersonUid(r.Context(), authStatus.PersonId)
	if err != nil {
		log.Printf("failed to get group UID for person: %v", authStatus.PersonId)
		err = layouts.RootPage(authStatus, nil, csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render root page template: %v", err)
		}
		return
	}

	grp, err := app.queries.GetGroupByUid(r.Context(), groupUID)
	if err != nil {
		log.Printf("failed to get group by UID %v: %v", groupUID, err)
		err = layouts.RootPage(authStatus, nil, csrf.Token(r)).Render(r.Context(), w)
		if err != nil {
			log.Printf("failed to render root page template: %v", err)
		}
		return
	}

	err = layouts.RootPage(authStatus, &grp, csrf.Token(r)).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render root page template: %v", err)
	}
}
