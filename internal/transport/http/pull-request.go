package http

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) handlePRCreate(w http.ResponseWriter, r *http.Request) {
	var req PRCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "invalid JSON",
			},
		})
		return
	}

	pr, err := h.prService.CreatePullRequest(r.Context(), req.ID, req.Name, req.Author)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, PRCreateResponse{
		PR: pullRequestToDto(pr),
	})
}

func (h *Handler) handlePRMerge(w http.ResponseWriter, r *http.Request) {
	var req PRMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "invalid JSON",
			},
		})
		return
	}

	pr, err := h.prService.MergePullRequest(r.Context(), req.ID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, PRMergeResponse{
		PR: pullRequestToDto(pr),
	})
}

func (h *Handler) handlePRReassign(w http.ResponseWriter, r *http.Request) {
	var req PRReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "invalid JSON",
			},
		})
		return
	}

	pr, replacedBy, err := h.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, PRReassignResponse{
		PR:         pullRequestToDto(pr),
		ReplacedBy: replacedBy,
	})
}
