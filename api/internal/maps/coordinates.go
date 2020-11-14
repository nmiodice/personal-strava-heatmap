package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

type mapStruct struct {
	Type string        `json:"type"`
	Data []interface{} `json:"data"`
}

func project(lat, lon float64) (float64, float64) {
	siny := math.Sin(lat * math.Pi / 180.0)
	siny = math.Min(math.Max(siny, -0.9999), 0.9999)
	x := tileSize * (0.5 + lon/360.0)
	y := tileSize * (0.5 - math.Log((1+siny)/(1-siny))/(4*math.Pi))
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

// https://gis.stackexchange.com/questions/17278/calculate-lat-lon-bounds-for-individual-tile-generated-from-gdal2tiles
func tileToLat(y, z int) float64 {
	n := math.Pi - 2*math.Pi*float64(y)/math.Pow(2, float64(z))
	return (180 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n))))
}

// https://gis.stackexchange.com/questions/17278/calculate-lat-lon-bounds-for-individual-tile-generated-from-gdal2tiles
func tileToLon(x, z int) float64 {
	return (float64(x)/math.Pow(2, float64(z))*360 - 180)
}
