package http

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	var teamDto TeamDTO
	if err := json.NewDecoder(r.Body).Decode(&teamDto); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "invalid JSON",
			},
		})
		return
	}

	team := teamFromDto(teamDto)
	created, err := h.teamsService.CreateTeam(r.Context(), team)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, TeamAddResponse{
		Team: teamToDto(created),
	})
}

func (h *Handler) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: errorBody{
				Code:    "BAD_REQUEST",
				Message: "team_name is required",
			},
		})
		return
	}

	team, err := h.teamsService.GetTeam(r.Context(), teamName)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, teamToDto(team))
}
