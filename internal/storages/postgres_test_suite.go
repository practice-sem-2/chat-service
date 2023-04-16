package storage

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PostgresTestSuite struct {
	suite.Suite
	db *sqlx.DB
	m  *migrate.Migrate
}

func (s *PostgresTestSuite) SetupSuite() {
	var err error
	viper.AutomaticEnv()
	dbDsn := viper.GetString("DB_DSN")
	migrationsDsn := viper.GetString("MIGRATIONS_DSN")
	migrationsDir := viper.GetString("MIGRATIONS_DIR")

	s.db, err = sqlx.Connect("pgx", dbDsn)
	require.NoError(s.T(), err, "failed to connect to database")

	s.m, err = migrate.New(migrationsDir, migrationsDsn)

	require.NoError(s.T(), err, "failed to open migrations")

	err = s.m.Up()
	require.NoError(s.T(), err, "failed to migrate database")
}
func (s *PostgresTestSuite) TearDownSuite() {
	_ = s.m.Down()
	_ = s.db.Close()
}
