package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sql/v2"

	"team-taskflow/internal/domain"
	"team-taskflow/internal/repository/mysql/mysqlerr"
)

type Repository struct {
	pool   *sql.DB
	getter *trmsql.CtxGetter
}

func NewRepository(pool *sql.DB) *Repository {
	return &Repository{pool: pool, getter: trmsql.DefaultCtxGetter}
}

func (r *Repository) Create(ctx context.Context, user domain.User) (int64, error) {
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

	result, err := executor.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		user.Email, user.PasswordHash, user.Name,
	)
	if err != nil {
		if mysqlerr.IsDuplicateEntry(err) {
			// The client-facing wording is the usecase's concern; the
			// repository only classifies the failure.
			return 0, fmt.Errorf("user with email %q: %w", user.Email, domain.ErrAlreadyExists)
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
	executor := r.getter.DefaultTrOrDB(ctx, r.pool)

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
