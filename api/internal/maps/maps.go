package maps

import (
	"context"
	"errors"
	"fmt"

	"github.com/nmiodice/personal-strava-heatmap/internal/batch"
	"github.com/nmiodice/personal-strava-heatmap/internal/concurrency"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
	"github.com/nmiodice/personal-strava-heatmap/internal/types"
)

const (
	TILE_SIZE = float64(256)
)

type MapService struct {
	stravaSvc               *strava.StravaService
	storageSvc              *storage.AzureBlobstore
	queueSvc                queue.QueueService
	minTileZoom             int
	maxTileZoom             int
	queueBatchSize          int
	storageConcurrencyLimit int
}

type mapStruct struct {
	Type string        `json:"type"`
	Data []interface{} `json:"data"`
}

type TileSet struct {
	tiles types.Set
}

type MapParam struct {
	FilenamePostfix string    `json:"postfix"`
	TopLeft         []float64 `json:"tl"`
	BottomRight     []float64 `json:"br"`
	Tile            Tile      `json:"tile"`
}

type MapParams []MapParam

func NewTileSet() TileSet {
	return TileSet{tiles: types.NewSet()}
}

type Tile struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

func (ts TileSet) Add(x, y, z int) {
	ts.tiles.Add(Tile{x, y, z})
}

func (ts TileSet) Size() int {
	return ts.tiles.Size()
}

func NewMapService(
	stravaSvc *strava.StravaService,
	storageSvc *storage.AzureBlobstore,
	queueSvc queue.QueueService,
	minTileZoom int,
	maxTileZoom int,
	queueBatchSize int,
	storageConcurrencyLimit int,
) *MapService {
	return &MapService{
		stravaSvc:               stravaSvc,
		storageSvc:              storageSvc,
		queueSvc:                queueSvc,
		minTileZoom:             minTileZoom,
		maxTileZoom:             maxTileZoom,
		queueBatchSize:          queueBatchSize,
		storageConcurrencyLimit: storageConcurrencyLimit,
	}
}

func (ms MapService) AddToTileSet(data []byte, minZoom, maxZoom int, tiles *TileSet) {
	coords := parseLatLonList(data)
	for z := minZoom; z <= maxZoom; z++ {
		scale := float64(int(1) << z)
		for _, coord := range coords {
			x, y := project(coord[0], coord[1])
			tiles.Add(
				int(x*scale/TILE_SIZE),
				int(y*scale/TILE_SIZE),
				z,
			)
		}
	}
}

func (ms MapService) ComputeMapParams(tiles *TileSet) MapParams {
	params := MapParams{}
	tileMap := tiles.tiles.ToMap()
	for k := range tileMap {
		tile := k.(Tile)
		fnPostfix := fmt.Sprintf("%d-%d-%d.png", tile.X, tile.Y, tile.Z)

		params = append(params, MapParam{
			FilenamePostfix: fnPostfix,
			TopLeft: []float64{
				tileToLat(tile.Y, tile.Z),
				tileToLon(tile.X, tile.Z),
			},
			BottomRight: []float64{
				tileToLat(tile.Y+1, tile.Z),
				tileToLon(tile.X+1, tile.Z),
			},
			Tile: tile,
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
	tiles := NewTileSet()

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

	mapParams := ms.ComputeMapParams(&tiles)
	messages := make([]interface{}, len(mapParams))
	for idx, p := range mapParams {
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

	if err = ms.queueSvc.Enqueue(ctx, messageBatches...); err != nil {
		return nil, nil, fmt.Errorf("%w: %+v", ErrorInternalError, err)
	}

	return dataRefs, messageBatches, nil
}
