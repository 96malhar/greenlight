package data

import (
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type MovieStoreInterface interface {
	// Insert a new record into the movies table.
	Insert(movie *Movie) error
	// Get a specific record from the movies table.
	Get(id int64) (*Movie, error)
	// Update a specific record in the movies table.
	Update(movie *Movie) error
	// Delete a specific record from the movies table.
	Delete(id int64) error
	// GetAll returns all movies from the movies table.
	GetAll(title string, genres []string, filters Filters) ([]*Movie, PaginationMetadata, error)
}

type UserStoreInterface interface {
	// Insert a new record into the users table.
	Insert(user *User) error
	// GetByEmail returns a specific record from the users table.
	GetByEmail(email string) (*User, error)
	// Update a specific record in the users table.
	Update(user *User) error
	// GetForToken retrieves a user record based on the token scope and plaintext token value.
	GetForToken(tokenScope, tokenPlaintext string) (*User, error)
}

type TokenStoreInterface interface {
	// New generates and stores a new token for a specific user and scope.
	New(userID int64, ttl time.Duration, scope string) (*Token, error)
	// Insert adds the data for a specific token to the tokens table.
	Insert(token *Token) error
	// DeleteAllForUser deletes all tokens for a specific user having a specific scope.
	DeleteAllForUser(scope string, userID int64) error
}

type PermissionStoreInterface interface {
	// GetAllForUser returns all permission codes for a specific user in a Permissions slice.
	GetAllForUser(userID int64) (Permissions, error)
	// AddForUser adds new permissions for a specific user.
	AddForUser(userID int64, codes ...string) error
}

type ModelStore struct {
	Movies      MovieStoreInterface
	Users       UserStoreInterface
	Tokens      TokenStoreInterface
	Permissions PermissionStoreInterface
}

func NewModelStore(db *pgxpool.Pool) ModelStore {
	return ModelStore{
		Movies:      MovieStore{db: db},
		Users:       UserStore{db: db},
		Tokens:      TokenStore{db: db},
		Permissions: PermissionStore{db: db},
	}
}
