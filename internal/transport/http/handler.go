package http

import (
	"context"
	"encoding/json"
	"net/http"

	"avito/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type TeamsService interface {
	CreateTeam(ctx context.Context, team domain.Team) (*domain.Team, error)
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
}

type UsersService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error)
}

type PullRequestsService interface {
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) (domain.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error)
}

type Handler struct {
	teamsService TeamsService
	usersService UsersService
	prService    PullRequestsService
}

func NewHandler(teams TeamsService, users UsersService, prs PullRequestsService) *Handler {
	return &Handler{
		teamsService: teams,
		usersService: users,
		prService:    prs,
	}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/team", func(r chi.Router) {
		r.Post("/add", h.handleTeamAdd)
		r.Get("/get", h.handleTeamGet)
	})

	router.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", h.handleUserSetIsActive)
		r.Get("/getReview", h.handleUsersGetReview)
	})

	router.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", h.handlePRCreate)
		r.Post("/merge", h.handlePRMerge)
		r.Post("/reassign", h.handlePRReassign)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return router
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status, body := mappingDomainErrors(err)
	writeJSON(w, status, body)
}
