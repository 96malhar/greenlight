package data

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

// Permissions is a slice of string, which we will use to hold the permission codes (like
// "movies:read" and "movies:write") for a single user.
type Permissions []string

// Include is a helper method which checks whether the Permissions slice contains a specific permission code.
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

type PermissionStore struct {
	db *pgxpool.Pool
}

// GetAllForUser returns all permission codes for a specific user in a Permissions slice.
func (s PermissionStore) GetAllForUser(userID int64) (Permissions, error) {
	query := `
        SELECT permissions.code
        FROM permissions
        INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
        INNER JOIN users ON users_permissions.user_id = users.id
        WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// AddForUser adds one or more permission codes for a specific user.
func (s PermissionStore) AddForUser(userID int64, codes ...string) error {
	query := `
        INSERT INTO users_permissions
        SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.db.Exec(ctx, query, userID, codes)
	return err
}
