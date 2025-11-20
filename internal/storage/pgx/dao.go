package pgx

import (
	"database/sql"
	"time"

	"avito/internal/domain"
)

type pullRequestDAO struct {
	ID        string
	Name      string
	AuthorID  string
	Status    string
	CreatedAt time.Time
	MergedAt  sql.NullTime
	Reviewers []string
}

func pullRequestDAOToDomain(pr pullRequestDAO) domain.PullRequest {
	var mergedAt *time.Time
	if pr.MergedAt.Valid {
		t := pr.MergedAt.Time
		mergedAt = &t
	} else {
		mergedAt = nil
	}

	return domain.PullRequest{
		ID:                pr.ID,
		Name:              pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            domain.PullRequestStatus(pr.Status),
		CreatedAt:         &pr.CreatedAt,
		MergedAt:          mergedAt,
		AssignedReviewers: pr.Reviewers,
	}
}
