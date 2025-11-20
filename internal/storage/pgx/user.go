package pgx

import (
	"context"
	"errors"

	"avito/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Storage) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	const query = `
		SELECT id, name, team_name, is_active
		  FROM users
		 WHERE id = $1;
	`

	var user domain.User
	err := s.getExecutor(ctx).QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.TeamName,
		&user.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *Storage) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	const query = `
		UPDATE users
		   SET is_active = $2
		 WHERE id = $1;
	`

	cmd, err := s.getExecutor(ctx).Exec(ctx, query, userID, isActive)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *Storage) ListActiveUserByTeam(ctx context.Context, teamName string) ([]domain.User, error) {
	const query = `
		SELECT id, name, team_name, is_active
		  FROM users
		 WHERE team_name = $1
		   AND is_active = true;
	`

	rows, err := s.getExecutor(ctx).Query(ctx, query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.TeamName,
			&user.IsActive,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
