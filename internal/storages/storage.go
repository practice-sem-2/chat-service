package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/jmoiron/sqlx"
)

type AtomicFunc func(Registry) error

type Registry interface {
	Atomic(ctx context.Context, fn AtomicFunc) error
	GetChatsStore() *ChatsStorage
	GetUpdatesStore() *UpdatesStorage
}

type DefaultRegistry struct {
	db       *sqlx.DB
	scope    Scope
	producer sarama.SyncProducer
	cfg      *UpdatesStoreConfig
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

func NewRegistry(db *sqlx.DB, p sarama.SyncProducer, cfg *UpdatesStoreConfig) *DefaultRegistry {
	return &DefaultRegistry{
		db:       db,
		scope:    db,
		producer: p,
		cfg:      cfg,
	}
}

func (r *DefaultRegistry) Atomic(ctx context.Context, fn AtomicFunc) (err error) {
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

	storage := DefaultRegistry{
		db:       r.db,
		scope:    tx,
		producer: r.producer,
		cfg:      r.cfg,
	}
	err = fn(&storage)
	return err
}

func (r *DefaultRegistry) GetChatsStore() *ChatsStorage {
	return NewChatsStorage(r.scope)
}

func (r *DefaultRegistry) GetUpdatesStore() *UpdatesStorage {
	return NewUpdatesStore(r.producer, r.cfg)
}
