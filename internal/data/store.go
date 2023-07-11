package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type MovieStoreInterface interface {
	Insert(movie *Movie) error
	Get(id int64) (*Movie, error)
	Update(movie *Movie) error
	Delete(id int64) error
}

type ModelStore struct {
	Movies MovieStoreInterface
}

func NewModelStore(db *sql.DB) ModelStore {
	return ModelStore{
		Movies: MovieStore{db: db},
	}
}
