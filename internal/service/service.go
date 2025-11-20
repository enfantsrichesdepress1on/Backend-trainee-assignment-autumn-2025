package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	"avito/internal/domain"
)

type TeamStorage interface {
	TeamExists(ctx context.Context, teamName string) (bool, error)
	CreateWithMembers(ctx context.Context, team domain.Team) error
	GetWithMembers(ctx context.Context, teamName string) (*domain.Team, error)
}

type UserStorage interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) error
	ListActiveUserByTeam(ctx context.Context, teamName string) ([]domain.User, error)
}

type PullRequestStorage interface {
	ListByReviewer(ctx context.Context, userID string) ([]domain.PullRequest, error)

	GetPullRequestByID(ctx context.Context, pullRequestID string) (domain.PullRequest, error)
	GetPullRequestByIDForUpdate(ctx context.Context, pullRequestID string) (domain.PullRequest, error)
	Create(ctx context.Context, pullRequest domain.PullRequest) error
	UpdateStatusMerged(ctx context.Context, pullRequestID string, mergedAt *time.Time) error
	ReplaceReviewer(ctx context.Context, pullRequestID string, oldID string, newID string) error
}

type txManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	teamStore TeamStorage
	userStore UserStorage
	prStore   PullRequestStorage
	tx        txManager
}

func NewService(teamStore TeamStorage, userStore UserStorage, prStore PullRequestStorage, tx txManager) *Service {
	return &Service{
		teamStore: teamStore,
		userStore: userStore,
		prStore:   prStore,
		tx:        tx,
	}
}

func (s *Service) CreateTeam(ctx context.Context, team domain.Team) (*domain.Team, error) {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		exists, err := s.teamStore.TeamExists(ctx, team.Name)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrTeamExists
		}

		if err := s.teamStore.CreateWithMembers(ctx, team); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.teamStore.GetWithMembers(ctx, team.Name)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	return s.teamStore.GetWithMembers(ctx, teamName)
}

func (s *Service) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	user, err := s.userStore.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.userStore.SetIsActive(ctx, userID, isActive); err != nil {
		return nil, err
	}

	user.IsActive = isActive
	return user, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	if _, err := s.userStore.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	return s.prStore.ListByReviewer(ctx, userID)
}

func (s *Service) CreatePullRequest(ctx context.Context, prID, prName, authorID string) (domain.PullRequest, error) {
	var created domain.PullRequest

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		_, err := s.prStore.GetPullRequestByID(ctx, prID)
		if err == nil {
			return domain.ErrPRExists
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		author, err := s.userStore.GetUserByID(ctx, authorID)
		if err != nil {
			return err
		}

		candidates, err := s.userStore.ListActiveUserByTeam(ctx, author.TeamName)
		if err != nil {
			return err
		}

		candidates = filterUsersExclude(candidates, []string{authorID})
		reviewers := chooseReviewers(candidates, 2)
		now := time.Now().UTC()

		pr := domain.PullRequest{
			ID:                prID,
			Name:              prName,
			AuthorID:          authorID,
			Status:            domain.PRStatusOpen,
			AssignedReviewers: reviewers,
			CreatedAt:         &now,
			MergedAt:          nil,
		}

		if err := s.prStore.Create(ctx, pr); err != nil {
			return err
		}

		created = pr
		return nil
	})

	if err != nil {
		return created, err
	}

	return created, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	var result domain.PullRequest

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		pr, err := s.prStore.GetPullRequestByIDForUpdate(ctx, prID)
		if err != nil {
			return err
		}

		if pr.Status == domain.PRStatusMerged { // idempotency
			result = pr
			return nil
		}

		now := time.Now().UTC()
		if err := s.prStore.UpdateStatusMerged(ctx, prID, &now); err != nil {
			return err
		}

		pr.Status = domain.PRStatusMerged
		pr.MergedAt = &now
		result = pr

		return nil
	})

	if err != nil {
		return result, err
	}

	return result, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error) {
	var (
		result     domain.PullRequest
		replacedBy string
	)

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		pr, err := s.prStore.GetPullRequestByIDForUpdate(ctx, prID)
		if err != nil {
			return err
		}

		if pr.Status == domain.PRStatusMerged {
			return domain.ErrPRMerged
		}

		if !slices.Contains(pr.AssignedReviewers, oldUserID) {
			return domain.ErrNotAssigned
		}

		oldUser, err := s.userStore.GetUserByID(ctx, oldUserID)
		if err != nil {
			return err
		}

		candidates, err := s.userStore.ListActiveUserByTeam(ctx, oldUser.TeamName)
		if err != nil {
			return err
		}
		exclude := append([]string{oldUserID, pr.AuthorID}, pr.AssignedReviewers...)
		candidates = filterUsersExclude(candidates, exclude)
		if len(candidates) == 0 {
			return domain.ErrNoCandidate
		}

		newReviewer := chooseReviewers(candidates, 1)[0]
		newID := newReviewer

		if err := s.prStore.ReplaceReviewer(ctx, prID, oldUserID, newID); err != nil {
			return err
		}

		for i, id := range pr.AssignedReviewers {
			if id == oldUserID {
				pr.AssignedReviewers[i] = newID
				break
			}
		}

		result = pr
		replacedBy = newID
		return nil
	})

	if err != nil {
		return result, replacedBy, err
	}

	return result, replacedBy, nil
}

func chooseReviewers(candidates []domain.User, quantity int) []string {
	if len(candidates) == 0 || quantity <= 0 {
		return nil
	}

	if quantity >= len(candidates) {
		outIDs := make([]string, 0, len(candidates))
		for _, user := range candidates {
			outIDs = append(outIDs, user.ID)
		}
		return outIDs
	}

	perm := rand.Perm(len(candidates))

	outIDs := make([]string, 0, quantity)
	for _, randIndex := range perm {
		if len(outIDs) == quantity {
			break
		}
		outIDs = append(outIDs, candidates[randIndex].ID)
	}

	return outIDs
}

func filterUsersExclude(users []domain.User, excludeIDs []string) []domain.User {
	if len(excludeIDs) == 0 {
		return users
	}
	excludesSet := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		excludesSet[id] = struct{}{}
	}

	out := make([]domain.User, 0, len(users))
	for _, user := range users {
		_, ok := excludesSet[user.ID]
		if ok {
			continue
		}
		out = append(out, user)
	}
	return out
}
