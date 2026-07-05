package teamrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sql/v2"
	"github.com/go-sql-driver/mysql"

	"team-taskflow/internal/domain"
)

const mysqlErrDuplicateEntry = 1062

type Repository struct {
	pool   *sql.DB
	getter *trmsql.CtxGetter
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool, getter: trmsql.DefaultCtxGetter}
}

func (r *Repository) CreateTeam(ctx context.Context, team domain.Team) (int64, error) {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		team.Name, team.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting team: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading inserted team id: %w", err)
	}
	return id, nil
}

func (r *Repository) GetTeam(ctx context.Context, teamID int64) (domain.Team, error) {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	var entity teamEntity
	err := executor.QueryRowContext(ctx,
		`SELECT id, name, created_by, created_at FROM teams WHERE id = ?`, teamID,
	).Scan(&entity.ID, &entity.Name, &entity.CreatedBy, &entity.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Team{}, fmt.Errorf("no team with id=%d: %w", teamID, domain.ErrNotFound)
		}
		return domain.Team{}, fmt.Errorf("selecting team: %w", err)
	}
	return entity.toDomain(), nil
}

func (r *Repository) AddMember(ctx context.Context, member domain.TeamMember) error {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	_, err := executor.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		member.TeamID, member.UserID, string(member.Role),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return fmt.Errorf("membership team=%d user=%d: %w", member.TeamID, member.UserID, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("inserting team member: %w", err)
	}
	return nil
}

func (r *Repository) GetMember(ctx context.Context, teamID, userID int64) (domain.TeamMember, error) {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	var entity memberEntity
	err := executor.QueryRowContext(ctx,
		`SELECT team_id, user_id, role, joined_at FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	).Scan(&entity.TeamID, &entity.UserID, &entity.Role, &entity.JoinedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TeamMember{}, fmt.Errorf("no membership team=%d user=%d: %w", teamID, userID, domain.ErrNotFound)
		}
		return domain.TeamMember{}, fmt.Errorf("selecting team member: %w", err)
	}
	return entity.toDomain(), nil
}

func (r *Repository) ListByUser(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	rows, err := executor.QueryContext(ctx,
		`SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = ?
		 ORDER BY t.id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("selecting teams by user: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.TeamWithRole
	for rows.Next() {
		var entity teamEntity
		var role string
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.CreatedBy, &entity.CreatedAt, &role); err != nil {
			return nil, fmt.Errorf("scanning team row: %w", err)
		}
		result = append(result, domain.TeamWithRole{Team: entity.toDomain(), Role: domain.Role(role)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating team rows: %w", err)
	}
	return result, nil
}
