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

func makeActivityStreamRefreshFunc(ctx context.Context, stravaSvc *strava.StravaService, lock locks.Lock) processor.ProcessorFunc {
	return func() error {
		athleteTokens, err := stravaSvc.Auth.GetAllCurrentAthleteAuthTokens(ctx)
		if err != nil {
			return err
		}

		log.Printf("will download activity streams for %d athletes", len(athleteTokens))
		var errors *multierror.Error
		for id, token := range athleteTokens {
			inserted, err := stravaSvc.Athlete.ImportMissingActivityStreams(ctx, token.AccessToken)
			if err != nil {
				errors = multierror.Append(errors, err)
			}

			log.Printf("downloaded '%d' new activity streams for athlete '%d'", inserted, id)

		}

		if errors != nil {
			return errors
		}

		return nil
	}
}

func AthleteActivityStreamRefreshConfig(ctx context.Context, stravaSvc *strava.StravaService, lock locks.Lock) processor.ProcessorConfiguration {
	return processor.ProcessorConfiguration{
		Func:     makeActivityStreamRefreshFunc(ctx, stravaSvc, lock),
		WaitTime: time.Hour * 1,
		Name:     "AthleteActivityStreamRefresh",
		Lock:     lock,
	}
}
