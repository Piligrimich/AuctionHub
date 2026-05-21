package postgres

import (
	"AuthService/internal/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (repo *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password)
		VALUES (:email, :password)
		RETURNING id, created_at, updated_at
	`
	rows, err := repo.db.NamedQueryContext(ctx, query, user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)

	if rows.Next() {
		return rows.StructScan(user)
	}
	return nil
}

func (repo *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := repo.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email=$1", email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("got user by email: %w", err)
	}
	return &user, nil
}

func (repo *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := repo.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id=$1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("got user by id: %w", err)
	}
	return &user, nil
}
