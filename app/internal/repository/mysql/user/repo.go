package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/infrastructure/tx"
)

const mysqlErrDuplicateEntry = 1062

type Repository struct {
	pool *sql.DB
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, user domain.User) (int64, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		user.Email, user.PasswordHash, user.Name,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return 0, fmt.Errorf("inserting user with email %q: %w", user.Email,
				&domain.SafeError{Kind: domain.ErrAlreadyExists, Msg: "user with this email already exists"})
		}
		return 0, fmt.Errorf("inserting user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading inserted user id: %w", err)
	}
	return id, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.getByField(ctx, "email", email)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	return r.getByField(ctx, "id", id)
}

func (r *Repository) getByField(ctx context.Context, field string, value any) (domain.User, error) {
	executor := tx.ExecutorFromContext(ctx, r.pool)

	var entity userEntity
	err := executor.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE %s = ?`, field),
		value,
	).Scan(&entity.ID, &entity.Email, &entity.PasswordHash, &entity.Name, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("no user with %s=%v: %w", field, value, domain.ErrNotFound)
		}
		return domain.User{}, fmt.Errorf("selecting user by %s: %w", field, err)
	}
	return entity.toDomain(), nil
}
