package domain

import "errors"

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrPRExists    = errors.New("pr already exists")
	ErrPRMerged    = errors.New("pr merged")
	ErrNotAssigned = errors.New("reviewer not assigned")
	ErrNoCandidate = errors.New("no candidate")
	ErrNotFound    = errors.New("not found")
)
