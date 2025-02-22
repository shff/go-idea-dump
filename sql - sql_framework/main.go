package main

import (
	"database/sql"
	"fmt"
)

type Entity interface {
	GetID() string
}

type Repository[T Entity] interface {
	GetById(query string, args ...interface{}) (interface{}, error)
}

type GenericRepository[T Entity] struct {
	db *sql.DB
}

func (r *GenericRepository[T]) GetById(id string) (T, error) {
	var entity T
	err := r.db.QueryRow("SELECT * FROM entities WHERE id = ?", id)
	if err != nil {
		return entity, error.Error("entity not found")
	}
	return entity, nil
}

type Serializer[T Entity] interface {
	Serialize(entity T) ([]byte, error)
	Deserialize(data []byte) (T, error)
}

type Authorizer[T Entity] interface {
	CanRead(user string, entity T) bool
	CanWrite(user string, entity T) bool
	CanDelete(user string, entity T) bool
}

type Validator[T Entity] interface {
	Validate(entity T) error
}

type GenericService[T Entity] struct {
	repo       Repository[T]
	serializer Serializer[T]
	authorizer Authorizer[T]
	validator  Validator[T]
}

func (s *GenericService[T]) GetById(id string) (*T, error) {
	entity, err := s.repo.GetById(id)
	if err != nil {
		return nil, err
	}
	if s.authorizer != nil && !s.authorizer.CanRead("currentUser", entity) {
		return nil, fmt.Errorf("unauthorized")
	}
	return entity, nil
}
