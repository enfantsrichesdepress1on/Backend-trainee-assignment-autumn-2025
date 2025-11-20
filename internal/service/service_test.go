package service

import (
	"avito/internal/domain"
	"avito/internal/service/mocks"
	"context"
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTxManager struct{}

func (f *mockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestService_MergePullRequest_Idempotent(t *testing.T) {
	ctx := context.Background()

	prStore := mocks.NewPullRequestStorage(t)
	userStore := mocks.NewUserStorage(t)
	teamStore := mocks.NewTeamStorage(t)
	tx := &mockTxManager{}

	pr := domain.PullRequest{
		ID:       "pr1",
		Status:   domain.PRStatusOpen,
		MergedAt: nil,
	}

	prStore.
		On("GetPullRequestByIDForUpdate", ctx, "pr1").
		Return(pr, nil).Once()

	prStore.
		On("UpdateStatusMerged", ctx, "pr1", mock.AnythingOfType("*time.Time")).
		Return(nil).Once()

	svc := NewService(teamStore, userStore, prStore, tx)

	got1, err1 := svc.MergePullRequest(ctx, "pr1")
	require.NoError(t, err1)
	assert.Equal(t, domain.PRStatusMerged, got1.Status)
	assert.NotNil(t, got1.MergedAt)

	mergedPR := got1
	prStore.
		On("GetPullRequestByIDForUpdate", ctx, "pr1").
		Return(mergedPR, nil).Once()

	got2, err2 := svc.MergePullRequest(ctx, "pr1")
	require.NoError(t, err2)
	assert.Equal(t, domain.PRStatusMerged, got2.Status)
	assert.Equal(t, got1.MergedAt, got2.MergedAt)

	prStore.AssertExpectations(t)
}

func TestService_ReassignReviewer_Success(t *testing.T) {
	ctx := context.Background()

	prStore := mocks.NewPullRequestStorage(t)
	userStore := mocks.NewUserStorage(t)
	teamStore := mocks.NewTeamStorage(t)
	tx := &mockTxManager{}

	pr := domain.PullRequest{
		ID:                "pr1",
		Status:            domain.PRStatusOpen,
		AuthorID:          "author",
		AssignedReviewers: []string{"r1", "r2"},
	}

	oldUser := &domain.User{
		ID:       "r1",
		TeamName: "team-A",
		IsActive: true,
	}

	candidates := []domain.User{
		{ID: "r1", TeamName: "team-A", IsActive: true},
		{ID: "r2", TeamName: "team-A", IsActive: true},
		{ID: "r3", TeamName: "team-A", IsActive: true},
	}

	prStore.
		On("GetPullRequestByIDForUpdate", ctx, "pr1").
		Return(pr, nil).Once()

	userStore.
		On("GetUserByID", ctx, "r1").
		Return(oldUser, nil).Once()

	userStore.
		On("ListActiveUserByTeam", ctx, "team-A").
		Return(candidates, nil).Once()

	prStore.
		On("ReplaceReviewer", ctx, "pr1", "r1", "r3").
		Return(nil).Once()

	svc := NewService(teamStore, userStore, prStore, tx)

	gotPR, replacedBy, err := svc.ReassignReviewer(ctx, "pr1", "r1")
	require.NoError(t, err)
	assert.Equal(t, "r3", replacedBy)
	assert.Equal(t, "pr1", gotPR.ID)
	assert.Equal(t, domain.PRStatusOpen, gotPR.Status)
	assert.Contains(t, gotPR.AssignedReviewers, "r3")
	assert.Contains(t, gotPR.AssignedReviewers, "r2")
	assert.NotContains(t, gotPR.AssignedReviewers, "r1")

	prStore.AssertExpectations(t)
	userStore.AssertExpectations(t)
}

func TestService_ReassignReviewer_NoCandidate(t *testing.T) {
	ctx := context.Background()

	prStore := mocks.NewPullRequestStorage(t)
	userStore := mocks.NewUserStorage(t)
	teamStore := mocks.NewTeamStorage(t)
	tx := &mockTxManager{}

	pr := domain.PullRequest{
		ID:                "pr1",
		Status:            domain.PRStatusOpen,
		AssignedReviewers: []string{"r1"},
	}

	oldUser := &domain.User{
		ID:       "r1",
		TeamName: "team-A",
		IsActive: true,
	}

	prStore.
		On("GetPullRequestByIDForUpdate", ctx, "pr1").
		Return(pr, nil).Once()

	userStore.
		On("GetUserByID", ctx, "r1").
		Return(oldUser, nil).Once()

	userStore.
		On("ListActiveUserByTeam", ctx, "team-A").
		Return([]domain.User{{ID: "r1", TeamName: "team-A", IsActive: true}}, nil).Once()

	svc := NewService(teamStore, userStore, prStore, tx)

	_, _, err := svc.ReassignReviewer(ctx, "pr1", "r1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNoCandidate))

	prStore.AssertExpectations(t)
	userStore.AssertExpectations(t)
}

func TestService_CreatePullRequest_AssignsUpToTwoReviewers(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		name             string
		activeUsers      []domain.User
		wantReviewersLen int
	}

	tests := []testCase{
		{
			name:             "no_candidates_assigns_zero",
			activeUsers:      []domain.User{},
			wantReviewersLen: 0,
		},
		{
			name: "one_candidate_assigns_one",
			activeUsers: []domain.User{
				{ID: "u2", TeamName: "team-A", IsActive: true},
			},
			wantReviewersLen: 1,
		},
		{
			name: "three_candidates_assigns_two",
			activeUsers: []domain.User{
				{ID: "u2", TeamName: "team-A", IsActive: true},
				{ID: "u3", TeamName: "team-A", IsActive: true},
				{ID: "u4", TeamName: "team-A", IsActive: true},
			},
			wantReviewersLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prStore := mocks.NewPullRequestStorage(t)
			userStore := mocks.NewUserStorage(t)
			teamStore := mocks.NewTeamStorage(t)
			tx := &mockTxManager{}

			author := &domain.User{
				ID:       "u1",
				TeamName: "team-A",
				IsActive: true,
			}

			prStore.
				On("GetPullRequestByID", ctx, "pr-1").
				Return(domain.PullRequest{}, domain.ErrNotFound).
				Once()

			userStore.
				On("GetUserByID", ctx, "u1").
				Return(author, nil).
				Once()

			users := append([]domain.User{*author}, tt.activeUsers...)
			userStore.
				On("ListActiveUserByTeam", ctx, "team-A").
				Return(users, nil).
				Once()

			prStore.
				On("Create", ctx, mock.MatchedBy(func(pr domain.PullRequest) bool {
					if pr.ID != "pr-1" || pr.AuthorID != "u1" {
						return false
					}
					if pr.Status != domain.PRStatusOpen {
						return false
					}
					if len(pr.AssignedReviewers) != tt.wantReviewersLen {
						return false
					}
					for _, id := range pr.AssignedReviewers {
						if id == "u1" {
							return false
						}
					}
					return true
				})).
				Return(nil).
				Once()

			svc := NewService(teamStore, userStore, prStore, tx)

			got, err := svc.CreatePullRequest(ctx, "pr-1", "PR name", "u1")
			require.NoError(t, err)
			assert.Equal(t, "pr-1", got.ID)
			assert.Equal(t, "u1", got.AuthorID)
			assert.Len(t, got.AssignedReviewers, tt.wantReviewersLen)

			prStore.AssertExpectations(t)
			userStore.AssertExpectations(t)
		})
	}
}

func Test_chooseReviewers(t *testing.T) {
	type args struct {
		candidates []domain.User
		quantity   int
	}

	tests := []struct {
		name          string
		args          args
		wantLen       int
		wantSubsetIDs []string
	}{
		{
			name: "no candidates",
			args: args{
				candidates: nil,
				quantity:   2,
			},
			wantLen:       0,
			wantSubsetIDs: nil,
		},
		{
			name: "quantity_zero",
			args: args{
				candidates: []domain.User{{ID: "u1"}, {ID: "u2"}},
				quantity:   0,
			},
			wantLen:       0,
			wantSubsetIDs: []string{"u1", "u2"},
		},
		{
			name: "quantity_more_than_candidates_returns_all",
			args: args{
				candidates: []domain.User{{ID: "u1"}, {ID: "u2"}},
				quantity:   5,
			},
			wantLen:       2,
			wantSubsetIDs: []string{"u1", "u2"},
		},
		{
			name: "quantity_less_than_candidates_returns_quantity_random_subset",
			args: args{
				candidates: []domain.User{{ID: "u1"}, {ID: "u2"}, {ID: "u3"}},
				quantity:   2,
			},
			wantLen:       2,
			wantSubsetIDs: []string{"u1", "u2", "u3"},
		},
	}

	idsSet := func(ids []string) map[string]struct{} {
		m := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			m[id] = struct{}{}
		}
		return m
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chooseReviewers(tt.args.candidates, tt.args.quantity)

			require.Equal(t, tt.wantLen, len(got), "unexpected len")

			if tt.wantSubsetIDs == nil {
				assert.Empty(t, got)
				return
			}

			allowed := idsSet(tt.wantSubsetIDs)
			seen := make(map[string]struct{}, len(got))

			for _, id := range got {
				_, ok := allowed[id]
				assert.True(t, ok, "unexpected id %q", id)

				_, dup := seen[id]
				assert.False(t, dup, "duplicate id %q", id)
				seen[id] = struct{}{}
			}
		})
	}
}

func Test_filterUsersExclude(t *testing.T) {
	users := []domain.User{
		{ID: "u1"},
		{ID: "u2"},
		{ID: "u3"},
	}

	tests := []struct {
		name       string
		excludeIDs []string
		wantIDs    []string
	}{
		{
			name:       "no_excludes_returns_all",
			excludeIDs: nil,
			wantIDs:    []string{"u1", "u2", "u3"},
		},
		{
			name:       "exclude_one",
			excludeIDs: []string{"u2"},
			wantIDs:    []string{"u1", "u3"},
		},
		{
			name:       "exclude_all",
			excludeIDs: []string{"u1", "u2", "u3"},
			wantIDs:    []string{},
		},
		{
			name:       "exclude_unknown",
			excludeIDs: []string{"u42"},
			wantIDs:    []string{"u1", "u2", "u3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterUsersExclude(users, tt.excludeIDs)
			gotIDs := make([]string, 0, len(got))
			for _, u := range got {
				gotIDs = append(gotIDs, u.ID)
			}
			assert.ElementsMatch(t, tt.wantIDs, gotIDs)
		})
	}
}
