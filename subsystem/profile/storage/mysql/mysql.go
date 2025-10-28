package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/subsystem/profile/storage/mysql/sqlc"
)

// Schema contains the MySQL schema for the DEP storage.
//
//go:embed schema.sql
var Schema string

// MySQLStorage implements a storage.AllStorage using MySQL.
type MySQLStorage struct {
	db *sql.DB
	q  *sqlc.Queries
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
//
// Default driver is "mysql".
// Value is ignored if WithDB is used.
func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// WithDB sets a custom MySQL *sql.DB to the storage.
//
// If set, driver passed via WithDriver is ignored.
func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

// New creates and returns a new MySQLStorage.
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
	return &MySQLStorage{db: cfg.db, q: sqlc.New(cfg.db)}, nil
}

const timestampFormat = "2006-01-02 15:04:05"

// RetrieveRawProfiles returns the raw profile bytes by name from MySQL.
// Implementations should not return all profiles if no names were provided.
// ErrProfileNotFound is returned for any name that hasn't been stored.
// ErrNoNames is returned if names is empty.
func (s *MySQLStorage) RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error) {
	if len(names) < 1 {
		return nil, storage.ErrNoNames
	}

	r, err := s.q.GetRawProfiles(ctx, names)
	if err != nil {
		return nil, err
	}

	ret := make(map[string][]byte)
	for _, dbrp := range r {
		ret[dbrp.Name] = dbrp.RawProfile
	}
	for _, name := range names {
		_, ok := ret[name]
		if !ok {
			return ret, fmt.Errorf("%w: %s: missing from result set", storage.ErrProfileNotFound, name)
		}
	}

	return ret, nil
}

// RetrieveProfileInfos returns the profile metadata by name from MySQL.
// ErrProfileNotFound is returned for any name that hasn't been stored.
func (s *MySQLStorage) RetrieveProfileInfos(ctx context.Context, names []string) (map[string]storage.ProfileInfo, error) {
	ret := make(map[string]storage.ProfileInfo)
	if len(names) > 0 {
		r, err := s.q.GetProfileInfos(ctx, names)
		if err != nil {
			return nil, err
		}

		for _, dbpi := range r {
			ret[dbpi.Name] = storage.ProfileInfo{
				Identifier: dbpi.ProfileID,
				UUID:       dbpi.ProfileUuid,
			}
		}
		for _, name := range names {
			_, ok := ret[name]
			if !ok {
				return ret, fmt.Errorf("%w: %s: missing from result set", storage.ErrProfileNotFound, name)
			}
		}
	} else {
		r, err := s.q.GetAllProfileInfos(ctx)
		if err != nil {
			return nil, err
		}

		for _, dbpi := range r {
			ret[dbpi.Name] = storage.ProfileInfo{
				Identifier: dbpi.ProfileID,
				UUID:       dbpi.ProfileUuid,
			}
		}
	}
	return ret, nil
}

// StoreProfile stores a raw profile and associated info in the profile storage by name from MySQL.
// It is up to the caller to make sure info is correctly populated and matches the raw profile bytes.
func (s *MySQLStorage) StoreProfile(ctx context.Context, name string, info storage.ProfileInfo, raw []byte) error {
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO subsystem_profiles 
	(name, profile_id, profile_uuid, raw_profile)
VALUES 
	(?, ?, ?, ?) as new
ON DUPLICATE KEY UPDATE 
	profile_id = new.profile_id,
	profile_uuid = new.profile_uuid,
	raw_profile = new.raw_profile;`,
		name,
		info.Identifier,
		info.UUID,
		raw,
	)
	return err
}

// DeleteProfile deletes a profile from profile storage by name from MySQL.
// ErrProfileNotFound is returned for a name that hasn't been stored.
func (s *MySQLStorage) DeleteProfile(ctx context.Context, name string) error {
	return s.q.DeleteProfile(ctx, name)
}
