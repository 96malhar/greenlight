package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type MovieStoreInterface interface {
	Insert(movie *Movie) error
	Get(id int64) (*Movie, error)
	Update(movie *Movie) error
	Delete(id int64) error
	GetAll(title string, genres []string, filters Filters) ([]*Movie, PaginationMetadata, error)
}

type UserStoreInterface interface {
	Insert(user *User) error
	GetByEmail(email string) (*User, error)
	Update(user *User) error
}

type ModelStore struct {
	Movies MovieStoreInterface
	Users  UserStoreInterface
}

func NewModelStore(db *sql.DB) ModelStore {
	return ModelStore{
		Movies: MovieStore{db: db},
		Users:  UserStore{db: db},
	}
}
