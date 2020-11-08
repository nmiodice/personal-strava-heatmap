package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	resty "github.com/go-resty/resty/v2"

	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
)

type Dependencies struct {
	HttpClient *resty.Client
	DB         *database.DB
	Strava     *strava.StravaService
	Map        *maps.MapService
	Storage    *storage.AzureBlobstore
	Queue      queue.QueueService
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

	http := &http.Client{Timeout: config.HttpClient.Timeout}
	restyClient := resty.
		NewWithClient(http).
		SetRetryCount(5).
		SetRetryWaitTime(500 * time.Millisecond).
		AddRetryCondition(func(r *resty.Response, e error) bool {
			if e != nil {
				return true
			}

			response := map[string]interface{}{}
			json.Unmarshal(r.Body(), &response)
			for _, key := range []string{"error", "Error", "errors", "Errors"} {
				if val, ok := response[key]; ok {
					log.Printf("Detected error in API response. Will retry: %+v", val)
					return true
				}
			}
			return false
		}).
		OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
			if r.StatusCode() >= 300 {
				log.Printf("Converting HTTP %d to error response", r.StatusCode())
				return fmt.Errorf("HTTP %s", r.Status())
			}
			return nil
		})

	athleteSvc := strava.NewAthleteService(restyClient, db, config.Strava.ConcurrencyLimit, storageClient)

	stravaService := &strava.StravaService{
		Auth: strava.NewOAuthService(
			restyClient,
			db,
			config.Strava.ClientID,
			config.Strava.ClientSecret,
		),
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
		DB:         db,
		HttpClient: restyClient,
		Strava:     stravaService,
		Map:        maps.NewMapService(),
		Storage:    storageClient,
		Queue:      queueService,
	}

	return deps, nil
}
