package database

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/ioc"
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
		err := s.tx.Commit()
		if err != nil {
			err = s.tx.Rollback()
			switch {
			case errors.Is(err, sql.ErrTxDone):
				return nil
			default:
				return fmt.Errorf("closing db transaction: %w", err)
			}
		}
	}

	return nil
}
