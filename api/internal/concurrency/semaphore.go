package concurrency

import (
	"github.com/hashicorp/go-multierror"
)

type empty struct{}

var emptySingleton = empty{}

// Semaphore Enables semaphore operations
type Semaphore struct {
	resources int
	ch        chan empty
}

// NewSemaphore creates a new semaphore with a specific resource count
func NewSemaphore(resources int) Semaphore {
	return Semaphore{resources, make(chan empty, resources)}
}

// Acquire acquire n resources
func (s Semaphore) Acquire(n int) {
	for i := 0; i < n; i++ {
		s.ch <- emptySingleton
	}
}

// Release release n resources
func (s Semaphore) Release(n int) {
	for i := 0; i < n; i++ {
		<-s.ch
	}
}

func (s Semaphore) WithRateLimit(funcs []func() error, stopOnError bool) error {
	errChan := make(chan error)
	doneChan := make(chan empty)

	for i := 0; i < len(funcs); i++ {
		go func(f func() error) {
			s.Acquire(1)
			defer s.Release(1)

			err := f()
			if err != nil {
				errChan <- err
				return
			}

			doneChan <- emptySingleton
		}(funcs[i])
	}

	totalDone := 0
	var result *multierror.Error
	for {
		select {
		case err := <-errChan:
			if stopOnError {
				return err
			}

			result = multierror.Append(result, err)
			totalDone++
			if totalDone == len(funcs) {
				return result.ErrorOrNil()
			}
		case <-doneChan:
			totalDone++
			if totalDone == len(funcs) {
				return result.ErrorOrNil()
			}
		}
	}
}
