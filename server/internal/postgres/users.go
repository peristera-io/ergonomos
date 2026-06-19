package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// CreateUser implements auth.Store. A duplicate email maps to auth.ErrEmailTaken.
func (db *DB) CreateUser(ctx context.Context, u auth.User) error {
	_, err := db.q(ctx).Exec(ctx,
		`INSERT INTO users (id, email, handle, instance_domain, is_local, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		string(u.Actor.ID), u.Email, u.Actor.Handle, u.Actor.Instance.Domain, u.Actor.Local, u.PasswordHash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return auth.ErrEmailTaken
		}
		return fmt.Errorf("postgres: create user: %w", err)
	}
	return nil
}

// FindByEmail implements auth.Store, returning auth.ErrNotFound when no row matches.
func (db *DB) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	var (
		u              auth.User
		id, handle     string
		instanceDomain string
		local          bool
	)
	err := db.q(ctx).QueryRow(ctx,
		`SELECT id, email, handle, instance_domain, is_local, password_hash
		 FROM users WHERE email = $1`, email,
	).Scan(&id, &u.Email, &handle, &instanceDomain, &local, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrNotFound
		}
		return auth.User{}, fmt.Errorf("postgres: find user by email: %w", err)
	}
	u.Actor = domain.Actor{
		ID:       domain.ID(id),
		Handle:   handle,
		Instance: domain.Instance{Domain: instanceDomain},
		Local:    local,
	}
	return u, nil
}
