package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Settings keys.
const (
	// MasterPasswordHashKey is the settings row holding the Private Archive hash.
	MasterPasswordHashKey = "master_password_hash"
	// UserFirstNameKey and UserLastNameKey hold the display name shown in the
	// header avatar and menu.
	UserFirstNameKey = "user_first_name"
	UserLastNameKey  = "user_last_name"
)

// DefaultUserFirstName and DefaultUserLastName are shown until the user sets
// their own name in Settings.
const (
	DefaultUserFirstName = "John"
	DefaultUserLastName  = "Doe"
)

// ErrMasterPasswordUnset is returned when no hash has been configured.
var ErrMasterPasswordUnset = errors.New("master password not configured")

// Profile is the user's display name.
type Profile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// FullName returns "First Last" with the parts that are set.
func (p Profile) FullName() string {
	switch {
	case p.FirstName != "" && p.LastName != "":
		return p.FirstName + " " + p.LastName
	case p.FirstName != "":
		return p.FirstName
	case p.LastName != "":
		return p.LastName
	default:
		return ""
	}
}

// GetProfile returns the stored profile, falling back to the default name.
func (r *SettingsRepository) GetProfile(ctx context.Context) (Profile, error) {
	values, err := r.getValues(ctx, UserFirstNameKey, UserLastNameKey)
	if err != nil {
		return Profile{}, err
	}

	profile := Profile{
		FirstName: values[UserFirstNameKey],
		LastName:  values[UserLastNameKey],
	}
	if profile.FirstName == "" && profile.LastName == "" {
		profile = Profile{FirstName: DefaultUserFirstName, LastName: DefaultUserLastName}
	}
	return profile, nil
}

// SetProfile stores the profile, treating blank fields as the default.
func (r *SettingsRepository) SetProfile(ctx context.Context, profile Profile) error {
	firstName := strings.TrimSpace(profile.FirstName)
	lastName := strings.TrimSpace(profile.LastName)
	if firstName == "" && lastName == "" {
		firstName = DefaultUserFirstName
		lastName = DefaultUserLastName
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for key, value := range map[string]string{
		UserFirstNameKey: firstName,
		UserLastNameKey:  lastName,
	} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO settings (key, value, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (key) DO UPDATE
			SET value = EXCLUDED.value, updated_at = now()`, key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return tx.Commit(ctx)
}

// getValues loads the given keys, omitting missing rows.
func (r *SettingsRepository) getValues(ctx context.Context, keys ...string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT key, value FROM settings WHERE key = ANY($1)`, keys)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	defer rows.Close()

	values := make(map[string]string, len(keys))
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

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
