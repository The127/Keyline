package database

import (
	"Keyline/ioc"
	"database/sql"
)

type DbService interface {
	GetTx() (*sql.Tx, error)
	Close() error
}

type dbService struct {
	tx *sql.Tx
	dp *ioc.DependencyProvider
}

func NewDbService(dp *ioc.DependencyProvider) DbService {
	return &dbService{
		dp: dp,
	}
}

func (s *dbService) GetTx() (*sql.Tx, error) {
	if s.tx != nil {
		return s.tx, nil
	}

	db := ioc.GetDependency[*sql.DB](s.dp)
	tx, err := db.Begin()
	s.tx = tx

	return tx, err
}

func (s *dbService) Close() error {
	if s.tx != nil {
		return s.tx.Commit()
	}

	// TODO: somehow get all errors/check if we had an error and rollback instead

	return nil
}
