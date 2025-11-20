package pgx

import (
	"context"
	"errors"
	"time"

	"avito/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Storage) Create(ctx context.Context, pr domain.PullRequest) error {
	const queryCreatePR = `
		INSERT INTO pull_requests (
		    id, name, author_id, status, created_at, merged_at
		) VALUES ($1, $2, $3, $4, $5, $6);
	`

	if pr.CreatedAt == nil {
		return errors.New("Create: pr.CreatedAt is nil")
	}

	var mergedAt any
	if pr.MergedAt != nil {
		mergedAt = *pr.MergedAt
	} else {
		mergedAt = nil
	}

	_, err := s.getExecutor(ctx).Exec(ctx, queryCreatePR, pr.ID, pr.Name, pr.AuthorID, pr.Status, *pr.CreatedAt, mergedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique
				return domain.ErrPRExists
			}
		}
		return err
	}

	const queryInsertReviewers = `
		INSERT INTO pull_request_reviewers (pull_request_id, user_id)
		VALUES ($1, $2);
	`

	for _, reviewerID := range pr.AssignedReviewers {
		if _, err := s.getExecutor(ctx).Exec(ctx, queryInsertReviewers, pr.ID, reviewerID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) GetPullRequestByID(ctx context.Context, pullRequestID string) (domain.PullRequest, error) {
	const query = `
		SELECT
		    p.id,
		    p.name,
		    p.author_id,
		    p.status,
		    p.created_at,
		    p.merged_at,
		    COALESCE(
		        array_agg(r.user_id) FILTER (WHERE r.user_id IS NOT NULL),
		        '{}'
		    ) AS reviewers
		  FROM pull_requests p
		  LEFT JOIN pull_request_reviewers r
		         ON r.pull_request_id = p.id
		 WHERE p.id = $1
		 GROUP BY p.id, p.name, p.author_id, p.status, p.created_at, p.merged_at;
	`

	var prDao pullRequestDAO
	err := s.getExecutor(ctx).QueryRow(ctx, query, pullRequestID).Scan(
		&prDao.ID,
		&prDao.Name,
		&prDao.AuthorID,
		&prDao.Status,
		&prDao.CreatedAt,
		&prDao.MergedAt,
		&prDao.Reviewers)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}

	return pullRequestDAOToDomain(prDao), nil
}

func (s *Storage) GetPullRequestByIDForUpdate(ctx context.Context, pullRequestID string) (domain.PullRequest, error) {
	const queryPR = `
		SELECT id, name, author_id, status, created_at, merged_at
		  FROM pull_requests
		 WHERE id = $1
		 FOR UPDATE;
	`

	var prDao pullRequestDAO
	err := s.getExecutor(ctx).QueryRow(ctx, queryPR, pullRequestID).Scan(
		&prDao.ID,
		&prDao.Name,
		&prDao.AuthorID,
		&prDao.Status,
		&prDao.CreatedAt,
		&prDao.MergedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}

	const queryReviewers = `
		SELECT user_id
		  FROM pull_request_reviewers
		 WHERE pull_request_id = $1;
	`

	rows, err := s.getExecutor(ctx).Query(ctx, queryReviewers, pullRequestID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer rows.Close()

	reviewers := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return domain.PullRequest{}, err
		}
		reviewers = append(reviewers, id)
	}
	if err := rows.Err(); err != nil {
		return domain.PullRequest{}, err
	}

	prDao.Reviewers = reviewers

	return pullRequestDAOToDomain(prDao), nil
}

func (s *Storage) UpdateStatusMerged(ctx context.Context, pullRequestID string, mergedAt *time.Time) error {
	const query = `
		UPDATE pull_requests
		   SET status   = $2,
		       merged_at = $3
		 WHERE id = $1;
	`

	if mergedAt == nil {
		return errors.New("mergedAt is nil in UpdateStatusMerged")
	}

	_, err := s.getExecutor(ctx).Exec(ctx, query,
		pullRequestID,
		string(domain.PRStatusMerged),
		*mergedAt,
	)
	return err
}

func (s *Storage) ReplaceReviewer(ctx context.Context, pullRequestID string, oldID string, newID string) error {
	const deleteQuery = `
		DELETE FROM pull_request_reviewers
		 WHERE pull_request_id = $1
		   AND user_id         = $2;
	`

	cmd, err := s.getExecutor(ctx).Exec(ctx, deleteQuery, pullRequestID, oldID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotAssigned
	}

	const insertQuery = `
		INSERT INTO pull_request_reviewers (pull_request_id, user_id)
		VALUES ($1, $2);
	`

	_, err = s.getExecutor(ctx).Exec(ctx, insertQuery, pullRequestID, newID)
	return err
}

func (s *Storage) ListByReviewer(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	const query = `
		SELECT
		    p.id,
		    p.name,
		    p.author_id,
		    p.status,
		    p.created_at,
		    p.merged_at,
		    COALESCE(
		        array_agg(r2.user_id) FILTER (WHERE r2.user_id IS NOT NULL),
		        '{}'
		    ) AS reviewers
		  FROM pull_requests p
		  JOIN pull_request_reviewers r
		    ON r.pull_request_id = p.id
		  LEFT JOIN pull_request_reviewers r2
		    ON r2.pull_request_id = p.id
		 WHERE r.user_id = $1
		 GROUP BY p.id, p.name, p.author_id, p.status, p.created_at, p.merged_at;
	`

	exec := s.getExecutor(ctx)

	rows, err := exec.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.PullRequest, 0)
	for rows.Next() {
		var dao pullRequestDAO

		if err := rows.Scan(
			&dao.ID,
			&dao.Name,
			&dao.AuthorID,
			&dao.Status,
			&dao.CreatedAt,
			&dao.MergedAt,
			&dao.Reviewers,
		); err != nil {
			return nil, err
		}

		out = append(out, pullRequestDAOToDomain(dao))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
