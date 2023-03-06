package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
)

type Registry struct {
	db    *sqlx.DB
	scope Scope
}

type Scope interface {
	sqlx.QueryerContext
	sqlx.ExecerContext
	sqlx.Execer
	sqlx.Queryer
	Get(dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExec(query string, arg interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
}

func NewRegistry(db *sqlx.DB) *Registry {
	return &Registry{
		db:    db,
		scope: db,
	}
}

func (r *Registry) Atomic(ctx context.Context, fn func(store *Registry) error) (err error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("rollback caused by error: \"%v\" failed: %v", err, rbErr)
			}
		} else {
			err = tx.Commit()
		}
	}()

	storage := Registry{r.db, tx}
	err = fn(&storage)
	return err
}

func (r *Registry) GetChatsStore() *ChatsStorage {
	return NewChatsStorage(r.scope)
}
