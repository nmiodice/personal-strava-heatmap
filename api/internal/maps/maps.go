package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"

	"github.com/nmiodice/personal-strava-heatmap/internal/types"
)

const (
	TILE_SIZE = float64(256)
)

type MapService struct {
}

type mapStruct struct {
	Type string        `json:"type"`
	Data []interface{} `json:"data"`
}

type TileSet struct {
	tiles types.Set
}

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

func NewMapService() *MapService {
	return &MapService{}
}

func (ms MapService) AddToTileSet(data []byte, minZoom, maxZoom int, tiles *TileSet) {
	coords := parseLatLonList(data)
	addTiles(coords, minZoom, maxZoom, tiles)
}

func addTiles(coords [][]float64, minZoom, maxZoom int, tiles *TileSet) {
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

func project(lat, lon float64) (float64, float64) {
	siny := math.Sin(lat * math.Pi / 180.0)
	siny = math.Min(math.Max(siny, -0.9999), 0.9999)
	x := TILE_SIZE * (0.5 + lon/360.0)
	y := TILE_SIZE * (0.5 - math.Log((1+siny)/(1-siny))/(4*math.Pi))
	return x, y
}

func parseLatLonList(data []byte) [][]float64 {
	results := [][]float64{}

	dec := json.NewDecoder(bytes.NewReader(data))

	// read open bracket
	dec.Token()

	// while the array contains values
	for dec.More() {
		var res mapStruct
		// decode an array value
		err := dec.Decode(&res)

		// skip non-conforming documents
		if err != nil {
			dec.Token()
			continue
		}

		// parse lat/lons
		switch res.Type {
		case "latlng":
			for _, dataElem := range res.Data {
				lat, lon, err := parseLatLon(dataElem)
				if err != nil {
					fmt.Println(err)
					continue
				}

				results = append(results, []float64{lat, lon})
			}
		}
	}

	return results
}

func parseLatLon(dataElem interface{}) (float64, float64, error) {
	asList, ok := dataElem.([]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("unexpectedly did not find lat/lon list: %+v", dataElem)
	}

	if len(asList) != 2 {
		return 0, 0, fmt.Errorf("unexpectedly did not find correct number of lat/lon: %+v", asList)
	}

	lat, ok := asList[0].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("lat was unexpectly not a float: %+v", asList[0])
	}
	lon, ok := asList[1].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("lon was unexpectly not a float: %+v", asList[0])
	}

	return lat, lon, nil
}

type MapParam struct {
	FilenamePostfix string    `json:"postfix"`
	TopLeft         []float64 `json:"tl"`
	BottomRight     []float64 `json:"br"`
	Tile            Tile      `json:"tile"`
}

type MapParams []MapParam

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

// https://gis.stackexchange.com/questions/17278/calculate-lat-lon-bounds-for-individual-tile-generated-from-gdal2tiles
func tileToLat(y, z int) float64 {
	n := math.Pi - 2*math.Pi*float64(y)/math.Pow(2, float64(z))
	return (180 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n))))
}

// https://gis.stackexchange.com/questions/17278/calculate-lat-lon-bounds-for-individual-tile-generated-from-gdal2tiles
func tileToLon(x, z int) float64 {
	return (float64(x)/math.Pow(2, float64(z))*360 - 180)
}
