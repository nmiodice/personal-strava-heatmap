package athlete

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/nmiodice/personal-strava-heatmap/internal/background/processor"
	"github.com/nmiodice/personal-strava-heatmap/internal/locks"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
)

func makeTokenRefreshFunc(ctx context.Context, stravaSvc *strava.StravaService, lock locks.Lock) processor.ProcessorFunc {
	return func() error {
		athleteTokens, err := stravaSvc.Auth.GetAllCurrentAthleteAuthTokens(ctx)
		if err != nil {
			return err
		}

		log.Printf("will update auth token for %d athletes", len(athleteTokens))
		var errors *multierror.Error
		for id := range athleteTokens {
			_, err := stravaSvc.Auth.RefreshAuthToken(ctx, id)
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		}

		if errors != nil {
			return errors
		}

		return nil
	}
}

func AthleteTokenRefreshConfig(ctx context.Context, stravaSvc *strava.StravaService, lock locks.Lock) processor.ProcessorConfiguration {
	return processor.ProcessorConfiguration{
		Func:     makeTokenRefreshFunc(ctx, stravaSvc, lock),
		WaitTime: time.Hour * 1,
		Name:     "AthleteTokenRefresh",
		Lock:     lock,
	}
}
