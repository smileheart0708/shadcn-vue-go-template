package database

import (
	"context"
	"fmt"
	"strings"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/database/sqlite"
	"main/internal/identity"
	"main/internal/setup"
)

const DriverSQLite = "sqlite"

type Config struct {
	Driver string
	DSN    string
}

// Runtime is the composition-root handle for a selected database adapter.
// Domain services receive their own repository instead of this aggregate.
type Runtime struct {
	driver          string
	close           func() error
	Identity        identity.Repository
	Sessions        auth.SessionRepository
	Setup           setup.Repository
	AccountPolicies accountpolicies.Repository
}

func Open(ctx context.Context, config Config) (*Runtime, error) {
	driver := strings.ToLower(strings.TrimSpace(config.Driver))
	if driver == "" {
		driver = DriverSQLite
	}

	switch driver {
	case DriverSQLite:
		store, err := sqlite.Open(ctx, config.DSN)
		if err != nil {
			return nil, fmt.Errorf("database: open SQLite adapter: %w", err)
		}
		return &Runtime{
			driver:          DriverSQLite,
			close:           store.Close,
			Identity:        store,
			Sessions:        store,
			Setup:           store,
			AccountPolicies: store,
		}, nil
	default:
		return nil, fmt.Errorf("database: driver %q is not compiled into this binary", driver)
	}
}

func (r *Runtime) Driver() string {
	if r == nil {
		return ""
	}
	return r.driver
}

func (r *Runtime) Close() error {
	if r == nil || r.close == nil {
		return nil
	}
	if err := r.close(); err != nil {
		return fmt.Errorf("database: close runtime: %w", err)
	}
	return nil
}
