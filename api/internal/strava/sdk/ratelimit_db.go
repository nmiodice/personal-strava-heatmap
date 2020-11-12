package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

type rateLimitDB struct {
	db *database.DB
}

func (rld rateLimitDB) UpdateLimittedUntilTime(ctx context.Context, limitUntil time.Time) error {
	return rld.db.InTx(ctx, pgx.ReadCommitted, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, updateLimittedUntilTimeSQL, limitUntil)
		var id bool
		if err := row.Scan(&id); err != nil {
			return fmt.Errorf("updating limited_until time: %w", err)
		}

		return nil
	})
}

func (rld rateLimitDB) GetLimittedUntilTime(ctx context.Context) time.Time {
	var limitUntil time.Time
	_ = rld.db.InTx(ctx, pgx.ReadCommitted, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, getLimitedUntilTimeSQL)

		// if an error occurrs, it probably indicates that there is no data in the table yet
		// so its safe to ignore. the zero value for time is suitable
		_ = row.Scan(&limitUntil)
		return nil
	})

	return limitUntil
}

var updateLimittedUntilTimeSQL = `
INSERT INTO
	StravaRateLimit
	(limited_until)
VALUES
	($1)
ON CONFLICT (id)
	DO UPDATE SET limited_until=EXCLUDED.limited_until, updated_at=NOW()
RETURNING
	id
`
var getLimitedUntilTimeSQL = `
SELECT
	limited_until
FROM
	StravaRateLimit
LIMIT
	1
`
