package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

type State string

const (
	ImportingActivities   State = "ImportingActivities"
	DownloadingActivities State = "DownloadingActivities"
	ComputingMapParams    State = "ComputingMapParams"
	ProcessingMap         State = "ProcessingMap"
)

func GetErrorState(errors []error) State {
	errorStrings := make([]string, len(errors))
	for i, err := range errors {
		errorStrings[i] = err.Error()
	}
	return State(fmt.Sprintf("Error::%d::%s", len(errorStrings), strings.Join(errorStrings, ",")))
}

type StateService interface {
	UpdateState(ctx context.Context, athleteID int, state State) error
	GetState(ctx context.Context, athleteID int) (*State, error)
}

type stateServiceImpl struct {
	db *database.DB
}

func NewStateService(db *database.DB) StateService {
	return stateServiceImpl{db}
}

func (s stateServiceImpl) UpdateState(ctx context.Context, athleteID int, state State) error {
	err := s.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, insertStateSQL, athleteID, state)

		var id int
		if err := row.Scan(&id); err != nil {
			return fmt.Errorf("setting state ID: %w", err)
		}

		return nil
	})

	return err
}

func (s stateServiceImpl) GetState(ctx context.Context, athleteID int) (*State, error) {
	var state *State

	err := s.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, "SELECT state FROM AthleteProcessingState WHERE athlete_id = $1", athleteID)
		stateString := ""

		if err := row.Scan(&stateString); err != nil && err != pgx.ErrNoRows {
			return fmt.Errorf("fetching state ID: %w", err)
		}

		s := State(stateString)
		state = &s

		return nil
	})

	return state, err
}

var insertStateSQL = `
INSERT INTO
	AthleteProcessingState
	(athlete_id, state)
VALUES
	($1, $2)
ON CONFLICT
	(athlete_id)
	DO UPDATE SET state=EXCLUDED.state, updated_at=NOW()
RETURNING
	athlete_id`
