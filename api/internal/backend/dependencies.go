package backend

import (
	"context"

	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/locks"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

type Dependencies struct {
	MakeLockFunc func(int) locks.Lock
	Strava       *strava.StravaService
	Map          *maps.MapService
}

func GetDependencies(ctx context.Context, config *Config) (*Dependencies, error) {
	db, err := database.NewDB(ctx, config.Database.ConnectionString())
	if err != nil {
		return nil, err
	}

	storageService, err := storage.NewAzureBlobstore(
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
		DB:           db,
	})
	athleteSvc := strava.NewAthleteService(stravaSDK, db, config.Strava.ConcurrencyLimit, storageService)

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

	mapSvc := maps.NewMapService(
		stravaService,
		storageService,
		queueService,
		db,
		config.Map.MinTileZoom,
		config.Map.MaxTileZoom,
		config.Queue.BatchSize,
		config.Storage.ConcurrencyLimit,
	)

	deps := &Dependencies{
		MakeLockFunc: func(id int) locks.Lock {
			return locks.NewDistributedLock(db, id)
		},
		Strava: stravaService,
		Map:    mapSvc,
	}

	return deps, nil
}
