package postgres

import (
	"AuthService/internal/domain"
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type refreshTokenRepo struct {
	db *sqlx.DB
}

func NewRefreshTokenRepo(db *sqlx.DB) *domain.RefreshTokenRepository {
	return &refreshTokenRepo{db: db}
}

func (repo *refreshTokenRepo) Create(ctx context.Context, token domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES (:user_id, :token_hash, :expires_at)
		RETURNING id, created_at
	`

	rows, err := repo.db.NamedQueryContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)

	if rows.Next() {
		return rows.StructScan(token)
	}
	return nil
}

func (repo *refreshTokenRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := repo.db.GetContext(ctx, &token, "SELECT * FROM refresh_tokens WHERE hash = $1", hash)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, domain.ErrInvalidToken
		}
	}
	return &token, nil
}

func (repo *refreshTokenRepo) DeleteByHash(ctx context.Context, hash string) error {
	_, err := repo.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE hash = $1", hash)
	return err
}
func (repo *refreshTokenRepo) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := repo.db.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	return err
}
