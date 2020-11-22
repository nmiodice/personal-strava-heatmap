package maps

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

const (
	processingQueued   = "QUEUED"
	processingComplete = "COMPLETE"
	processingFailed   = "FAILED"
)

type mapDB struct {
	db *database.DB
}

func (mdb mapDB) setProcessingStateForIDs(ctx context.Context, mapID string, ids []string) error {
	queryArgs := []interface{}{}
	idx := 1
	queryFormat := ""

	for _, id := range ids {
		if queryFormat != "" {
			queryFormat += ", "
		}
		queryArgs = append(queryArgs, mapID, id, processingQueued)
		queryFormat += fmt.Sprintf("($%d, $%d, $%d)", idx, idx+1, idx+2)
		idx += 3
	}

	query := fmt.Sprintf(insertProcessingStateForIDsSQL, queryFormat)
	err := mdb.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, queryArgs...)
		rows.Close()
		return err
	})
	return err
}

type ProcessingState struct {
	Queued   int
	Failed   int
	Complete int
}

func (mdb mapDB) getProcessingStateForMap(ctx context.Context, mapID string) (*ProcessingState, error) {
	var pstate ProcessingState

	err := mdb.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, getProcessingStateForMapSQL, mapID)
		if err := row.Scan(&pstate.Queued, &pstate.Failed, &pstate.Complete); err != nil && err != pgx.ErrNoRows {
			return fmt.Errorf("fetching processing states form ap: %w", err)
		}
		return nil
	})
	return &pstate, err
}

// substitution is a series of escaped SQL values blocks
var insertProcessingStateForIDsSQL = `
INSERT INTO
	QueueProcessingState
	(map_id, message_id, pstate)
VALUES
	%s
`

// group by each state, filtering on the latest insertion date
var getProcessingStateForMapSQL = `
SELECT
	count(message_id) FILTER (WHERE pstate='` + processingQueued + `')   AS "queued",
    count(message_id) FILTER (WHERE pstate='` + processingFailed + `')   AS "failed",
    count(message_id) FILTER (WHERE pstate='` + processingComplete + `') AS "complete"
FROM
	QueueProcessingState
WHERE
	created_at=(SELECT MAX(created_at) FROM QueueProcessingState)
    AND
    map_id = $1
GROUP BY
	map_id
;
`
