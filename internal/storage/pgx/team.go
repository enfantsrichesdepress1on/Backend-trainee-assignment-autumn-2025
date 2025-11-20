package pgx

import (
	"context"
	"errors"

	"avito/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (s *Storage) TeamExists(ctx context.Context, teamName string) (bool, error) {
	const query = `select name from teams where name = $1;`

	var name string
	err := s.getExecutor(ctx).QueryRow(ctx, query, teamName).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateWithMembers Must use in business layer, only with tx
func (s *Storage) CreateWithMembers(ctx context.Context, team domain.Team) error {
	const queryTeam = `insert into teams (name) values ($1);`

	_, err := s.getExecutor(ctx).Exec(ctx, queryTeam, team.Name)
	if err != nil {
		return err
	}

	const queryUser = `insert into users (id, name, team_name, is_active) values ($1, $2, $3, $4);`
	for _, member := range team.Members {
		_, err := s.getExecutor(ctx).Exec(ctx, queryUser, member.ID, member.Name, team.Name, member.IsActive)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) GetWithMembers(ctx context.Context, teamName string) (*domain.Team, error) {
	const queryTeam = `select name from teams where name = $1;`

	var name string
	err := s.getExecutor(ctx).QueryRow(ctx, queryTeam, teamName).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	const queryUser = `select id, name, is_active from users where team_name = $1;`

	rows, err := s.getExecutor(ctx).Query(ctx, queryUser, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.User, 0)
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.IsActive); err != nil {
			return nil, err
		}
		user.TeamName = teamName
		members = append(members, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &domain.Team{
		Name:    name,
		Members: members,
	}, nil
}
