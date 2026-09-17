package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepository stores application-level secrets in the settings table.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository constructs a SettingsRepository.
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// MasterPasswordHashKey is the settings row holding the Private Archive hash.
const MasterPasswordHashKey = "master_password_hash"

// ErrMasterPasswordUnset is returned when no hash has been configured.
var ErrMasterPasswordUnset = errors.New("master password not configured")

// GetMasterPasswordHash returns the stored bcrypt hash.
func (r *SettingsRepository) GetMasterPasswordHash(ctx context.Context) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx,
		`SELECT value FROM settings WHERE key = $1`, MasterPasswordHashKey,
	).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || value == "" {
			return "", ErrMasterPasswordUnset
		}
		return "", fmt.Errorf("get master password hash: %w", err)
	}
	if value == "" {
		return "", ErrMasterPasswordUnset
	}
	return value, nil
}

// SetMasterPasswordHash inserts or replaces the stored hash.
func (r *SettingsRepository) SetMasterPasswordHash(ctx context.Context, hash string) error {
	if hash == "" {
		return fmt.Errorf("master password hash must not be empty")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()`,
		MasterPasswordHashKey, hash)
	if err != nil {
		return fmt.Errorf("set master password hash: %w", err)
	}
	return nil
}
