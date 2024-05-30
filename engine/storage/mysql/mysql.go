package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/micromdm/nanocmd/engine/storage/mysql/sqlc"
)

const mySQLTimestampFormat = "2006-01-02 15:04:05"

// MySQLStorage implements a storage.AllStorage using MySQL.
type MySQLStorage struct {
	db *sql.DB
	q  *sqlc.Queries

	randMu sync.Mutex
	rand   *rand.Rand
}

type config struct {
	driver string
	dsn    string
	db     *sql.DB
}

// Option allows configuring a MySQLStorage.
type Option func(*config)

// WithDSN sets the storage MySQL data source name.
func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

// WithDriver sets a custom MySQL driver for the storage.
// Default driver is "mysql" but is ignored if WithDB is used.
func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// WithDB sets a custom MySQL *sql.DB to the storage.
// If set, driver passed via WithDriver is ignored.
func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

// New creates and returns a new MySQL.
func New(opts ...Option) (*MySQLStorage, error) {
	cfg := &config{driver: "mysql"}
	for _, opt := range opts {
		opt(cfg)
	}
	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(cfg.driver, cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLStorage{
		db:   cfg.db,
		q:    sqlc.New(cfg.db),
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// sqlNullString sets Valid to true of the return value of s is not empty.
func sqlNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// sqlNullTime sets Valid to true of the return value of t is not zero.
func sqlNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Valid: !t.IsZero(), Time: t}
}

// txcb executes SQL within transactions when wrapped in tx().
type txcb func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error

// tx wraps g in transactions using db.
// If g returns an err the transaction will be rolled back; otherwise committed.
func tx(ctx context.Context, db *sql.DB, q *sqlc.Queries, g txcb) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx begin: %w", err)
	}
	if err = g(ctx, tx, q.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %w; while trying to handle error: %v", rbErr, err)
		}
		return fmt.Errorf("tx rolled back: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}
