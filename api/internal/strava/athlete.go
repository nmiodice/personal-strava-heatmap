package strava

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
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

func (as AthleteService) ImportMissingActivityStreams(ctx context.Context, token string) (int, error) {
	athleteID, err := as.oauthDB.getAthleteForAuthToken(ctx, token)
	if err != nil {
		return 0, err
	}

	unsyncedActivities, err := as.athleteDB.UnsyncedActivities(ctx, athleteID)
	if err != nil {
		return 0, err
	}

	count := 0
	var errors *multierror.Error
	for _, activityID := range unsyncedActivities {
		theActivity := activityID
		err := as.syncSingleActivity(ctx, token, athleteID, theActivity)
		if err != nil {
			errors = multierror.Append(errors, err)
		} else {
			count++
		}
		log.Printf("downloaded %d of %d activities", count, len(unsyncedActivities))
	}

	return count, errors
}

func (as AthleteService) syncSingleActivity(ctx context.Context, token string, athleteID int, activityID int64) error {
	activity, err := as.stravaSDK.GetActivityBytes(ctx, token, activityID)
	if err == sdk.ErrorNotFound {
		return nil
	}
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
