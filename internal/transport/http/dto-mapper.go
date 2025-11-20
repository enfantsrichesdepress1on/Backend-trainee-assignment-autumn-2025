package http

import (
	"errors"
	"net/http"

	"avito/internal/domain"
)

func teamFromDto(team TeamDTO) domain.Team {
	members := make([]domain.User, len(team.Members))
	for i, member := range team.Members {
		members[i] = domain.User{
			ID:       member.UserID,
			Name:     member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
	}
	return domain.Team{
		Name:    team.TeamName,
		Members: members,
	}
}

func teamToDto(t *domain.Team) TeamDTO {
	members := make([]TeamMemberDTO, 0, len(t.Members))
	for _, member := range t.Members {
		members = append(members, TeamMemberDTO{
			UserID:   member.ID,
			Username: member.Name,
			IsActive: member.IsActive,
		})
	}

	return TeamDTO{
		TeamName: t.Name,
		Members:  members,
	}
}

func userToDto(user *domain.User) UserDTO {
	return UserDTO{
		UserID:   user.ID,
		Username: user.Name,
		IsActive: user.IsActive,
		TeamName: user.TeamName,
	}
}

func pullRequestShortToDto(pr domain.PullRequest) PullRequestShortDTO {
	return PullRequestShortDTO{
		ID:       pr.ID,
		Name:     pr.Name,
		AuthorID: pr.AuthorID,
		Status:   string(pr.Status),
	}
}

func pullRequestToDto(pr domain.PullRequest) PullRequestDTO {
	reviewers := make([]string, len(pr.AssignedReviewers))
	copy(reviewers, pr.AssignedReviewers)
	return PullRequestDTO{
		ID:                pr.ID,
		Name:              pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: reviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}

func mappingDomainErrors(err error) (int, ErrorResponse) {
	var code string
	var status int

	switch {
	case errors.Is(err, domain.ErrTeamExists):
		status = http.StatusBadRequest
		code = "TEAM_EXISTS"

	case errors.Is(err, domain.ErrPRExists):
		status = http.StatusConflict
		code = "PR_EXISTS"

	case errors.Is(err, domain.ErrPRMerged):
		status = http.StatusConflict
		code = "PR_MERGED"

	case errors.Is(err, domain.ErrNotAssigned):
		status = http.StatusConflict
		code = "NOT_ASSIGNED"

	case errors.Is(err, domain.ErrNoCandidate):
		status = http.StatusConflict
		code = "NO_CANDIDATE"

	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "NOT_FOUND"

	default:
		status = http.StatusInternalServerError
		code = "INTERNAL"
	}

	return status, ErrorResponse{
		Error: errorBody{
			Code:    code,
			Message: err.Error(),
		},
	}
}
