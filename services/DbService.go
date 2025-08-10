package services

import (
	"Keyline/ioc"
	"database/sql"
)

type DbService struct {
	tx *sql.Tx
	dp *ioc.DependencyProvider
}

func NewDbService(dp *ioc.DependencyProvider) *DbService {
	return &DbService{
		dp: dp,
	}
}

func (s *DbService) GetTx() (*sql.Tx, error) {
	if s.tx != nil {
		return s.tx, nil
	}

	db := ioc.GetDependency[*sql.DB](s.dp)
	tx, err := db.Begin()
	s.tx = tx

	return tx, err
}

func (s *DbService) Close() error {
	if s.tx != nil {
		return s.tx.Commit()
	}

	// TODO: somehow get all errors/check if we had an error and rollback instead

	return nil
}
