package storage

import (
	"github.com/jackc/pgconn"
)

func GetPgxConstraintName(err error) string {
	if err == nil {
		return ""
	}
	pgErr, ok := err.(*pgconn.PgError)

	if !ok {
		return ""
	}

	return pgErr.ConstraintName
}
