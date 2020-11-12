package processor

import (
	"context"
	"log"
	"time"

	"github.com/nmiodice/personal-strava-heatmap/internal/locks"
)

type ProcessorFunc func() error
type ProcessorConfiguration struct {
	Func     ProcessorFunc
	WaitTime time.Duration
	Name     string
	Lock     locks.Lock
}

func runLoop(ctx context.Context, config ProcessorConfiguration) {
	var theFunc ProcessorFunc = config.Func
	if config.Lock != nil {
		theFunc = wrapFuncWithLock(ctx, config.Func, config.Lock)
	}

	for {
		err := theFunc()
		if err != nil {
			log.Printf("Processing failed for configuration %s: %+v", config.Name, err)
		}

		time.Sleep(config.WaitTime)
	}
}

func wrapFuncWithLock(ctx context.Context, pFunc ProcessorFunc, lock locks.Lock) ProcessorFunc {
	return func() error {
		gotLock, err := lock.WithLock(ctx, pFunc)

		log.Printf("GOT_LOCK: %+v | GOT_ERROR: %+v", gotLock, err)
		return err
	}
}

func RunForever(ctx context.Context, config ProcessorConfiguration) {
	go runLoop(ctx, config)
}
