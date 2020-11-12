package sdk

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

const (
	rateLimitHeader      = "X-Ratelimit-Limit"
	rateLimitUsageHeader = "X-Ratelimit-Usage"
)

// determine whether or not to retry a request
func retryConditionFunc(r *resty.Response, err error) bool {
	switch err {
	case ErrorInternalServerError:
		log.Printf("Recieved %+v (%s), will retry", err, r.Status())
		return true
	default:
		return false
	}
}

// convert non 200 status code responses into error
func afterResponseConvertNon200ToError(c *resty.Client, r *resty.Response) error {
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

// fail request in rate limit exceeded condition
func makeAPILimitRequestMiddleware(db *rateLimitDB) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		limitedUntil := db.GetLimittedUntilTime(r.Context())
		if time.Now().UTC().Before(limitedUntil) {
			log.Printf("artificially rate limiting until %+v based on rate limit table metadata", limitedUntil)
			return ErrorTooManyRequests
		}
		return nil
	}
}

type rateLimit struct {
	fifteenMinute int
	daily         int
}

// parse header containing rate limit information
func parseRateLimitHeader(r *resty.Response, headerName string) *rateLimit {
	h := r.Header().Get(headerName)
	parts := strings.Split(h, ",")
	if len(parts) != 2 {
		return nil
	}

	fifteenMinute, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	daily, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}

	return &rateLimit{fifteenMinute, daily}
}

func getDelayTime(bucket time.Duration) time.Time {
	return time.Now().UTC().Truncate(bucket).Add(bucket)
}

// persist consumed rate limit
func makeAPILimitResponseMiddleware(db *rateLimitDB) resty.ResponseMiddleware {
	return func(c *resty.Client, r *resty.Response) error {
		limits := parseRateLimitHeader(r, rateLimitHeader)
		used := parseRateLimitHeader(r, rateLimitUsageHeader)
		if limits == nil || used == nil {
			return nil
		}

		var limitUntil time.Time
		if used.daily >= limits.daily {
			limitUntil = getDelayTime(time.Hour * 24)
		} else if used.fifteenMinute >= limits.fifteenMinute {
			limitUntil = getDelayTime(time.Minute * 15)
		}

		err := db.UpdateLimittedUntilTime(r.Request.Context(), limitUntil)
		if err != nil {
			log.Printf("Error updating delay timestamp for strava client: %+v", err)
		}

		return nil
	}
}

func newHTTPClient(timeout time.Duration, db *database.DB) *resty.Client {
	http := &http.Client{Timeout: timeout}
	rlDB := &rateLimitDB{db}
	return resty.
		NewWithClient(http).
		SetRetryCount(5).
		SetRetryWaitTime(500 * time.Millisecond).
		AddRetryCondition(retryConditionFunc).
		OnBeforeRequest(makeAPILimitRequestMiddleware(rlDB)).
		OnAfterResponse(makeAPILimitResponseMiddleware(rlDB)).
		OnAfterResponse(afterResponseConvertNon200ToError)
}
