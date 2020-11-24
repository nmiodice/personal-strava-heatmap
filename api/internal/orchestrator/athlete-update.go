package orchestrator

import (
	"context"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/state"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
)

// UpdateAthleteMap update a map for an athlete, and track the progress.
// This function processes optimistically and will continue in spite of errors
func UpdateAthleteMap(
	stravaSvc *strava.StravaService,
	mapSvc *maps.MapService,
	stateSvc state.StateService,
	athleteID int,
	accessToken string,
	ctx context.Context) error {

	var errors *multierror.Error

	log.Printf("importing new activities for athlete '%d'", athleteID)
	stateSvc.UpdateState(ctx, athleteID, state.ImportingActivities)

	_, err := stravaSvc.Athlete.ImportNewActivities(ctx, accessToken)
	if err != nil {
		multierror.Append(errors, err)
		log.Printf("error encountered importing new activities for athlete '%d': %+v", athleteID, err)
	}

	log.Printf("importing new activity streams for athlete '%d'", athleteID)
	stateSvc.UpdateState(ctx, athleteID, state.DownloadingActivities)

	imported, err := stravaSvc.Athlete.ImportMissingActivityStreams(ctx, accessToken)
	if err != nil {
		multierror.Append(errors, err)
		log.Printf("error encountered importing new activity streams for athlete '%d': %+v", athleteID, err)
	}

	if imported > 0 {
		log.Printf("rebuilding map for athlete '%d'", athleteID)
		stateSvc.UpdateState(ctx, athleteID, state.ComputingMapParams)

		dataRefs, messageBatches, err := mapSvc.RebuildMapForAthlete(ctx, accessToken)
		if err != nil {
			log.Printf("error encountered rebuilding map for athlete '%d': %+v", athleteID, err)
			multierror.Append(errors, err)
		} else {
			log.Printf("rebuilt map using '%d' data refs and '%d' queued messages for athlete '%d'", len(dataRefs), len(messageBatches), athleteID)
		}
	}

	stateSvc.UpdateState(ctx, athleteID, state.ProcessingMap)
	if errors != nil && len(errors.Errors) > 0 {
		stateSvc.UpdateState(ctx, athleteID, state.GetErrorState(errors.Errors))
		return errors
	}

	return nil
}
