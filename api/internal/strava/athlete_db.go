package strava

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

type athleteDB struct {
	db *database.DB
}

func (ad athleteDB) InsertActivities(ctx context.Context, activities []sdk.Activity) ([]int64, error) {
	inserted := []int64{}
	queryArgs := []interface{}{}
	idx := 1
	queryFormat := ""

	for _, activity := range activities {
		if queryFormat != "" {
			queryFormat += ", "
		}
		queryArgs = append(queryArgs, activity.Athlete.ID, activity.ID, nil)
		queryFormat += fmt.Sprintf("($%d, $%d, $%d)", idx, idx+1, idx+2)

		idx += 3
	}

	query := fmt.Sprintf(insertActivitiesSQL, queryFormat)

	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, queryArgs...)
		if err != nil {
			return err
		}

		defer rows.Close()
		for rows.Next() {
			var id int64
			err = rows.Scan(&id)
			if err != nil {
				return err
			}

			inserted = append(inserted, id)
		}

		return nil
	})
	return inserted, err
}

func (ad athleteDB) UnsyncedActivities(ctx context.Context, athleteID int) ([]int64, error) {
	unprocessed := []int64{}
	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, unsyncedActivitiesSQL, athleteID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			err := rows.Scan(&id)
			if err != nil {
				return err
			}

			unprocessed = append(unprocessed, id)
		}

		return nil
	})
	return unprocessed, err
}

func (ad athleteDB) UpdateActivityWithDataRef(ctx context.Context, athleteID int, activityID int64, dataRef string) error {
	return ad.db.InTx(ctx, pgx.RepeatableRead, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, updateActivityWithDataRefSQL, athleteID, activityID, dataRef)
		var updatedActivityID int64
		if err := row.Scan(&updatedActivityID); err != nil {
			return fmt.Errorf("fetching activity_id for updated raw data: %w", err)
		}

		return nil
	})
}

func (ad athleteDB) GetActivityDataRefs(ctx context.Context, athleteID int) ([]string, error) {
	dataRefs := []string{}

	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, syncedActivityDataRefSQL, athleteID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ref string
			err := rows.Scan(&ref)
			if err != nil {
				return err
			}

			dataRefs = append(dataRefs, ref)
		}

		return nil
	})

	return dataRefs, err
}

func (ad athleteDB) GetOrCreateMapID(ctx context.Context, athleteID int) (string, error) {
	mapID := ""
	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, insertOrGetMapIDSQL, athleteID)

		if err := row.Scan(&mapID); err != nil {
			return fmt.Errorf("fetching map ID: %w", err)
		}

		return nil
	})

	return mapID, err
}

func (ad athleteDB) SetMapSharable(ctx context.Context, mapID string) error {
	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, setMapSharableSQL, mapID)

		if err := row.Scan(&mapID); err != nil {
			return fmt.Errorf("setting map share status: %w", err)
		}

		return nil
	})

	return err
}

func (ad athleteDB) GetMapSharable(ctx context.Context, mapID string) (bool, error) {
	sharable := false
	err := ad.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, getMapSharableSQL, mapID)

		if err := row.Scan(&sharable); err != nil {
			return fmt.Errorf("fetching map share status: %w", err)
		}

		return nil
	})

	return sharable, err
}

// substitution is a series of escaped SQL values blocks
var insertActivitiesSQL = `
INSERT INTO
	StravaActivity
	(athlete_id, activity_id, activity_data_ref)
VALUES
	%s
ON CONFLICT (activity_id)
	DO NOTHING
RETURNING
	activity_id
`

var unsyncedActivitiesSQL = `
SELECT
	activity_id
FROM
	StravaActivity
WHERE
	athlete_id = $1
		AND
	(activity_data_ref IS NULL OR activity_data_ref = '')
`

var syncedActivityDataRefSQL = `
SELECT
	activity_data_ref
FROM
	StravaActivity
WHERE
	athlete_id = $1
		AND
	(activity_data_ref IS NOT NULL AND activity_data_ref <> '')
`

var updateActivityWithDataRefSQL = `
UPDATE
	StravaActivity
SET
	activity_data_ref = $3
WHERE
	athlete_id = $1 AND activity_id = $2
RETURNING
	activity_id
`

var insertOrGetMapIDSQL = `
INSERT INTO
	AthleteMap
	(athlete_id)
VALUES
	($1)
ON CONFLICT (athlete_id)
	DO UPDATE SET athlete_id=EXCLUDED.athlete_id
RETURNING
	id
`

var setMapSharableSQL = `
UPDATE
	AthleteMap
SET
	sharable = true
WHERE
	id = $1
RETURNING
	id
`

var getMapSharableSQL = `
SELECT
	sharable
FROM
	AthleteMap
WHERE
	id = $1
`
