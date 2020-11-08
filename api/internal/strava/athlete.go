package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	resty "github.com/go-resty/resty/v2"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/concurrency"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
)

type athleteClient struct {
	httpClient *resty.Client
}

type athleteDB struct {
	db *database.DB
}

type AthleteService struct {
	concurrencyLimit int
	client           athleteClient
	db               athleteDB
	oauthDB          oauthDB
	storageClient    *storage.AzureBlobstore
}

func NewAthleteService(httpClient *resty.Client, db *database.DB, concurrencyLimit int, storageClient *storage.AzureBlobstore) *AthleteService {
	return &AthleteService{
		concurrencyLimit: concurrencyLimit,
		client: athleteClient{
			httpClient: httpClient,
		},
		db: athleteDB{
			db: db,
		},
		oauthDB: oauthDB{
			db: db,
		},
		storageClient: storageClient,
	}
}

func (as AthleteService) GetAthleteForAuthToken(ctx context.Context, token string) (int, error) {
	return as.oauthDB.getAthleteForAuthToken(ctx, token)
}

func (as AthleteService) GetActivityDataRefs(ctx context.Context, token string) ([]string, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return as.db.GetActivityDataRefs(ctx, athleteID)
}

func (as AthleteService) RefreshActivities(ctx context.Context, token string) (*ActivitySummary, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return nil, err
	}

	activities, err := as.client.ListAllActivities(ctx, token)
	if err != nil {
		return nil, err
	}

	newActivities, err := as.db.InsertActivities(ctx, activities)
	if err != nil {
		return nil, err
	}

	unsyncedActivities, err := as.db.UnsyncedActivities(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	return &ActivitySummary{
		Total:    len(activities),
		New:      newActivities,
		Unsynced: unsyncedActivities,
	}, nil
}

func (as AthleteService) SyncActivities(ctx context.Context, token string) (int, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return 0, err
	}

	unsyncedActivities, err := as.db.UnsyncedActivities(ctx, athleteID)
	if err != nil {
		return 0, err
	}

	count := 0
	countSem := concurrency.NewSemaphore(1)
	funcs := make([](func() error), len(unsyncedActivities))
	for i, activityID := range unsyncedActivities {
		theActivity := activityID
		funcs[i] = func() error {
			err := as.syncSingleActivity(ctx, token, athleteID, theActivity)
			if err != nil {
				return err
			}

			time.Sleep(1 * time.Second) // agressive rate limit on Strava API...
			countSem.Acquire(1)
			defer countSem.Release(1)
			count++
			log.Printf("Downloaded %d of %d activities", count, len(unsyncedActivities))
			return nil
		}
	}

	err = concurrency.NewSemaphore(as.concurrencyLimit).WithRateLimit(funcs, false)
	return count, err
}

func (as AthleteService) syncSingleActivity(ctx context.Context, token string, athleteID int, activityID int64) error {
	activity, err := as.client.GetActivity(ctx, token, activityID)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d/%d.json", athleteID, activityID)
	err = as.storageClient.CreateObject(ctx, fileName, activity)
	if err != nil {
		return err
	}

	err = as.db.UpdateActivityWithDataRef(ctx, athleteID, activityID, fileName)
	if err != nil {
		return err
	}

	return nil
}

func (as AthleteService) GetOrCreateMapID(ctx context.Context, token string) (string, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return "", err
	}

	return as.db.GetOrCreateMapID(ctx, athleteID)
}

func (ac athleteClient) ListAllActivities(ctx context.Context, token string) ([]Activity, error) {
	page := 1
	activities := []Activity{}
	for {
		pageActivities, err := ac.getActivityIDsFromPage(ctx, token, page)
		if err != nil {
			return activities, err
		}

		if len(pageActivities) == 0 {
			return activities, nil
		}

		activities = append(activities, pageActivities...)
		page++
	}
}

func (ac athleteClient) getActivityIDsFromPage(ctx context.Context, token string, page int) ([]Activity, error) {
	activities := []Activity{}

	res, err := ac.httpClient.R().
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParams(map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": "100",
		}).
		Get("https://www.strava.com/api/v3/activities")

	if err != nil {
		return activities, err
	}

	err = json.Unmarshal(res.Body(), &activities)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

func (ac athleteClient) GetActivity(ctx context.Context, token string, activityID int64) ([]byte, error) {
	url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d/streams?keys=latlng", activityID)
	res, err := ac.httpClient.R().
		SetHeader("Authorization", "Bearer "+token).
		Get(url)

	if err != nil {
		return nil, err
	}
	return res.Body(), nil
}

func (ad athleteDB) InsertActivities(ctx context.Context, activities []Activity) ([]int64, error) {
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
	log.Printf("%d | %d", athleteID, activityID)
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
