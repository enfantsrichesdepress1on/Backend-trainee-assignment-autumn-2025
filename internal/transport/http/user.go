package http

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) handleUserSetIsActive(w http.ResponseWriter, r *http.Request) {
	var req UserSetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "invalid JSON",
			},
		})
		return
	}

	user, err := h.usersService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, UserSetIsActiveResponse{
		User: userToDto(user),
	})
}

func (h *Handler) handleUsersGetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "user_id is required",
			},
		})
		return
	}

	prs, err := h.usersService.GetUserReviews(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := UsersGetReviewResponse{
		UserID:       userID,
		PullRequests: make([]PullRequestShortDTO, 0, len(prs)),
	}

	for _, pr := range prs {
		resp.PullRequests = append(resp.PullRequests, pullRequestShortToDto(pr))
	}

	writeJSON(w, http.StatusOK, resp)
}
