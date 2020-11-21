package athlete

import (
	"context"
	"log"
	"time"

	"github.com/nmiodice/personal-strava-heatmap/internal/background/processor"
	"github.com/nmiodice/personal-strava-heatmap/internal/locks"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/orchestrator"
	"github.com/nmiodice/personal-strava-heatmap/internal/state"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
)

func makeAthleteUpdateFunc(ctx context.Context, stravaSvc *strava.StravaService, mapService *maps.MapService, stateService state.StateService, lock locks.Lock) processor.ProcessorFunc {
	return func() error {
		athleteTokens, err := stravaSvc.Auth.GetAllCurrentAthleteAuthTokens(ctx)
		if err != nil {
			return err
		}

		log.Printf("updating activity information for %d athletes", len(athleteTokens))
		for athleteID, token := range athleteTokens {
			orchestrator.UpdateAthleteMap(
				stravaSvc,
				mapService,
				stateService,
				athleteID,
				token.AccessToken,
				context.Background())
		}

		return nil
	}
}

func AthleteUpdateConfig(ctx context.Context, stravaSvc *strava.StravaService, mapService *maps.MapService, stateService state.StateService, lock locks.Lock) processor.ProcessorConfiguration {
	return processor.ProcessorConfiguration{
		Func:     makeAthleteUpdateFunc(ctx, stravaSvc, mapService, stateService, lock),
		WaitTime: time.Hour * 1,
		Name:     "AthleteUpdate",
		Lock:     lock,
	}
}
