package athlete

import (
	"fmt"
	"log"
	"time"

	"github.com/nmiodice/personal-strava-heatmap/internal/background/processor"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
)

func makeActivityRefreshFunc(stravaSvc *strava.StravaService) processor.ProcessorFunc {
	count := 0
	return func() (processor.ProcessorResults, error) {
		count++
		if count%2 == 0 {
			return nil, fmt.Errorf("foo")
		}
		log.Println("looper")
		return processor.ProcessorResults{
			"foo":  "bar",
			"foo2": 1,
		}, nil
	}
}

func ActivityRefreshConfig(stravaSvc *strava.StravaService) processor.ProcessorConfiguration {
	return processor.ProcessorConfiguration{
		Func:     makeActivityRefreshFunc(stravaSvc),
		WaitTime: time.Second * 5,
		Name:     "Activity Refresh",
	}
}
