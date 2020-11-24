package locks

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

type Lock interface {
	WithLock(ctx context.Context, f func() error) (bool, error)
}

type lockImpl struct {
	db     *database.DB
	lockID int
}

// NewDistributedLock returns a database backed distributed lock
func NewDistributedLock(db *database.DB, lockID int) Lock {
	return lockImpl{db, lockID}
}

func (l lockImpl) WithLock(ctx context.Context, f func() error) (bool, error) {
	var acquiredLock bool
	err := l.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", l.lockID)
		if err := row.Scan(&acquiredLock); err != nil {
			return fmt.Errorf("error trying to acquire lock: %w", err)
		}
		return nil
	})

	if err == nil && acquiredLock {
		err = f()
	}
	return acquiredLock, err
}
