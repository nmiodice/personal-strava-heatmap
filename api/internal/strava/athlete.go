package strava

import (
	"context"
	"fmt"
	"log"

	"github.com/nmiodice/personal-strava-heatmap/internal/concurrency"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

type AthleteService struct {
	concurrencyLimit int
	stravaSDK        sdk.StravaSDK
	athleteDB        athleteDB
	oauthDB          oauthDB
	storageClient    *storage.AzureBlobstore
}

func NewAthleteService(stravaSDK sdk.StravaSDK, db *database.DB, concurrencyLimit int, storageClient *storage.AzureBlobstore) *AthleteService {
	return &AthleteService{
		concurrencyLimit: concurrencyLimit,
		stravaSDK:        stravaSDK,
		athleteDB: athleteDB{
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
	return as.athleteDB.GetActivityDataRefs(ctx, athleteID)
}

func (as AthleteService) ImportNewActivities(ctx context.Context, token string) (int, error) {
	activities, err := as.stravaSDK.ListAllActivities(ctx, token)
	if err != nil {
		return 0, err
	}

	newActivities, err := as.athleteDB.InsertActivities(ctx, activities)
	if err != nil {
		return 0, err
	}

	return len(newActivities), nil
}

type ActivitySyncSummary struct {
	Total   int
	Sunc    int
	NotSunc int
	Errored int
}

func (as AthleteService) ImportMissingActivityStreams(ctx context.Context, token string) (int, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return 0, err
	}

	unsyncedActivities, err := as.athleteDB.UnsyncedActivities(ctx, athleteID)
	if err != nil {
		return 0, err
	}

	successCount := 0
	notFoundCount := 0
	countSem := concurrency.NewSemaphore(1)
	funcs := make([](func() error), len(unsyncedActivities))
	for i, activityID := range unsyncedActivities {
		theActivity := activityID
		funcs[i] = func() error {
			err := as.syncSingleActivity(ctx, token, athleteID, theActivity)
			if err != nil && err != sdk.ErrorNotFound {
				return err
			}

			countSem.Acquire(1)
			defer countSem.Release(1)

			if err == sdk.ErrorNotFound {
				notFoundCount++
				log.Printf("activity '%d' for athlete '%d' was not found", activityID, athleteID)
			} else {
				successCount++
				log.Printf("downloaded '%d' of '%d' activitiesfor athlete '%d'", successCount+notFoundCount, len(unsyncedActivities), athleteID)
			}

			return nil
		}
	}

	err = concurrency.NewSemaphore(as.concurrencyLimit).WithRateLimit(funcs, false)
	return successCount, err
}

func (as AthleteService) syncSingleActivity(ctx context.Context, token string, athleteID int, activityID int64) error {
	activity, err := as.stravaSDK.GetActivityBytes(ctx, token, activityID)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d/%d.json", athleteID, activityID)
	err = as.storageClient.CreateObject(ctx, fileName, activity)
	if err != nil {
		return err
	}

	err = as.athleteDB.UpdateActivityWithDataRef(ctx, athleteID, activityID, fileName)
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

	return as.athleteDB.GetOrCreateMapID(ctx, athleteID)
}

func (as AthleteService) SetMapSharable(ctx context.Context, mapID string) error {
	return as.athleteDB.SetMapSharable(ctx, mapID)
}

func (as AthleteService) GetMapSharable(ctx context.Context, mapID string) (bool, error) {
	return as.athleteDB.GetMapSharable(ctx, mapID)
}
