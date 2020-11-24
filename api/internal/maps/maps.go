package maps

import (
	"context"
	"errors"
	"fmt"

	"github.com/nmiodice/personal-strava-heatmap/internal/batch"
	"github.com/nmiodice/personal-strava-heatmap/internal/concurrency"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
	"github.com/nmiodice/personal-strava-heatmap/internal/types"
)

const (
	tileSize = float64(256)
)

type MapService struct {
	stravaSvc               *strava.StravaService
	storageSvc              *storage.AzureBlobstore
	queueSvc                queue.QueueService
	db                      *mapDB
	minTileZoom             int
	maxTileZoom             int
	queueBatchSize          int
	storageConcurrencyLimit int
}

type MapParam struct {
	FilenamePostfix string    `json:"postfix"`
	TopLeft         []float64 `json:"tl"`
	BottomRight     []float64 `json:"br"`
	Tile            Tile      `json:"tile"`
}

type mapParams []MapParam

type Tile struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type tileSet struct {
	tiles types.Set
}

func (ts tileSet) Add(x, y, z int) {
	ts.tiles.Add(Tile{x, y, z})
}

func (ts tileSet) Size() int {
	return ts.tiles.Size()
}

func NewMapService(
	stravaSvc *strava.StravaService,
	storageSvc *storage.AzureBlobstore,
	queueSvc queue.QueueService,
	db *database.DB,
	minTileZoom int,
	maxTileZoom int,
	queueBatchSize int,
	storageConcurrencyLimit int,
) *MapService {
	return &MapService{
		stravaSvc:               stravaSvc,
		storageSvc:              storageSvc,
		queueSvc:                queueSvc,
		db:                      &mapDB{db},
		minTileZoom:             minTileZoom,
		maxTileZoom:             maxTileZoom,
		queueBatchSize:          queueBatchSize,
		storageConcurrencyLimit: storageConcurrencyLimit,
	}
}

func (ms MapService) AddToTileSet(data []byte, minZoom, maxZoom int, tiles *tileSet) {
	coords := parseLatLonList(data)
	for z := minZoom; z <= maxZoom; z++ {
		scale := float64(int(1) << z)
		for _, coord := range coords {
			x, y := project(coord[0], coord[1])
			tiles.Add(
				int(x*scale/tileSize),
				int(y*scale/tileSize),
				z,
			)
		}
	}
}

func (ms MapService) ComputeMapParams(tiles *tileSet) mapParams {
	params := mapParams{}
	tileMap := tiles.tiles.ToMap()
	for k := range tileMap {
		t := k.(Tile)
		fnPostfix := fmt.Sprintf("%d-%d-%d.png", t.X, t.Y, t.Z)

		params = append(params, MapParam{
			FilenamePostfix: fnPostfix,
			TopLeft: []float64{
				tileToLat(t.Y, t.Z),
				tileToLon(t.X, t.Z),
			},
			BottomRight: []float64{
				tileToLat(t.Y+1, t.Z),
				tileToLon(t.X+1, t.Z),
			},
			Tile: t,
		})
	}

	return params
}

var (
	ErrorMissingToken  = errors.New("missing authentication token in reuqest")
	ErrorStravaAPI     = errors.New("encountered issue with Strava API")
	ErrorInternalError = errors.New("encountered issue with backend subsystem")
)

func (ms MapService) RebuildMapForAthlete(ctx context.Context, token string) ([]string, []interface{}, error) {
	if token == "" {
		return nil, nil, ErrorMissingToken
	}

	dataRefs, err := ms.stravaSvc.Athlete.GetActivityDataRefs(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %+v", ErrorStravaAPI, err)
	}

	mapSem := concurrency.NewSemaphore(1)
	tiles := tileSet{types.NewSet()}

	funcs := [](func() error){}
	for _, ref := range dataRefs {
		theRef := ref
		funcs = append(funcs, func() error {
			bytes, err := ms.storageSvc.GetObjectBytes(ctx, theRef)
			if err != nil {
				return fmt.Errorf("%w: %+v", ErrorInternalError, err)
			}

			mapSem.Acquire(1)
			defer mapSem.Release(1)

			ms.AddToTileSet(bytes, ms.minTileZoom, ms.maxTileZoom, &tiles)
			return nil
		})
	}

	if err = concurrency.NewSemaphore(ms.storageConcurrencyLimit).WithRateLimit(funcs, true); err != nil {
		return nil, nil, err
	}

	params := ms.ComputeMapParams(&tiles)
	messages := make([]interface{}, len(params))
	for idx, p := range params {
		messages[idx] = p
	}

	athleteID, err := ms.stravaSvc.Athlete.GetAthleteForAuthToken(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %+v", ErrorStravaAPI, err)
	}

	mapID, err := ms.stravaSvc.Athlete.GetOrCreateMapID(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %+v", ErrorStravaAPI, err)
	}

	messageBatches := batch.ToBatchesWithTransformer(messages, ms.queueBatchSize, func(batch []interface{}) interface{} {
		return map[string]interface{}{
			"coords":     batch,
			"athlete_id": athleteID,
			"map_id":     mapID,
		}
	})

	messageIDs, err := ms.queueSvc.Enqueue(ctx, messageBatches...)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %+v", ErrorInternalError, err)
	}

	err = ms.db.setProcessingStateForIDs(ctx, mapID, messageIDs)
	return dataRefs, messageBatches, err
}

func (ms MapService) GetProcessingStateForAthlete(ctx context.Context, token string) (*ProcessingState, error) {
	mapID, err := ms.stravaSvc.Athlete.GetOrCreateMapID(ctx, token)
	if err != nil {
		return nil, err
	}

	return ms.db.getProcessingStateForMap(ctx, mapID)
}
