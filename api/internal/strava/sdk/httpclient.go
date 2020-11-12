package sdk

import (
	"log"
	"net/http"
	"time"

	resty "github.com/go-resty/resty/v2"
)

func onAfterResponseFunc(c *resty.Client, r *resty.Response) error {
	switch r.StatusCode() {
	case http.StatusBadRequest:
		return ErrorBadRequest
	case http.StatusUnauthorized:
		return ErrorUnauthorized
	case http.StatusNotFound:
		return ErrorNotFound
	case http.StatusTooManyRequests:
		return ErrorTooManyRequests
	case http.StatusInternalServerError:
		return ErrorInternalServerError
	default:
		if r.StatusCode() >= 300 {
			return makeHTTPError(r.StatusCode())
		}
		return nil
	}
}

func retryConditionFunc(r *resty.Response, err error) bool {
	if err == ErrorInternalServerError {
		log.Printf("Recieved %+v (%s), will retry", err, r.Status())
		return true
	}

	if err != nil {
		log.Printf("Recieved %+v (%s), will not retry", err, r.Status())
		return true
	}

	return false
}

func newHTTPClient(timeout time.Duration) *resty.Client {
	http := &http.Client{Timeout: timeout}
	restyClient := resty.
		NewWithClient(http).
		SetRetryCount(5).
		SetRetryWaitTime(500 * time.Millisecond).
		AddRetryCondition(retryConditionFunc).
		OnAfterResponse(onAfterResponseFunc)

	return restyClient
}
