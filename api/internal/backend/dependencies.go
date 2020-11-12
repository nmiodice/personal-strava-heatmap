package backend

import (
	"context"

	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

type Dependencies struct {
	DB      *database.DB
	Strava  *strava.StravaService
	Map     *maps.MapService
	Storage *storage.AzureBlobstore
	Queue   queue.QueueService
}

func GetDependencies(ctx context.Context, config *Config) (*Dependencies, error) {
	db, err := database.NewDB(ctx, config.Database.ConnectionString())
	if err != nil {
		return nil, err
	}

	storageClient, err := storage.NewAzureBlobstore(
		ctx,
		config.Storage.ContainerName,
		config.Storage.AccountName,
		config.Storage.AccountKey)
	if err != nil {
		return nil, err
	}

	stravaSDK := sdk.NewStravaSDK(sdk.StravaSDKConfig{
		Timeout:      config.HttpClient.Timeout,
		ClientID:     config.Strava.ClientID,
		ClientSecret: config.Strava.ClientSecret,
	})
	athleteSvc := strava.NewAthleteService(stravaSDK, db, config.Strava.ConcurrencyLimit, storageClient)

	stravaService := &strava.StravaService{
		Auth:    strava.NewOAuthService(stravaSDK, db),
		Athlete: athleteSvc,
	}

	queueService, err := queue.NewAzureStorageQueue(
		ctx,
		config.Queue.QueueName,
		config.Queue.AccountName,
		config.Queue.AccountKey)

	if err != nil {
		return nil, err
	}

	deps := &Dependencies{
		DB:      db,
		Strava:  stravaService,
		Map:     maps.NewMapService(),
		Storage: storageClient,
		Queue:   queueService,
	}

	return deps, nil
}
