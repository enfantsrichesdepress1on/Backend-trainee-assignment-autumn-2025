package domain

import "time"

type User struct {
	ID       string
	Name     string
	TeamName string
	IsActive bool
}

type Team struct {
	ID      string
	Name    string
	Members []User
}

type PullRequestStatus string

const (
	PRStatusOpen   PullRequestStatus = "OPEN"
	PRStatusMerged PullRequestStatus = "MERGED"
)

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PullRequestStatus
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}
