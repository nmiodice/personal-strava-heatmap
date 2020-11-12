package processor

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type ProcessorResults map[string]interface{}
type ProcessorFunc func() (ProcessorResults, error)
type ProcessorConfiguration struct {
	Func     ProcessorFunc
	WaitTime time.Duration
	Name     string
}

func runLoop(ctx context.Context, config ProcessorConfiguration) {
	for {
		results, err := config.Func()
		if err != nil {
			log.Printf("Processing failed for configuration %s: %+v", config.Name, err)
		} else {
			jsonBytes, _ := json.Marshal(results)
			log.Printf("Processing complete for %s. Results: %s", config.Name, string(jsonBytes))
		}

		time.Sleep(config.WaitTime)
	}
}

func RunForever(ctx context.Context, config ProcessorConfiguration) {
	go runLoop(ctx, config)
}
