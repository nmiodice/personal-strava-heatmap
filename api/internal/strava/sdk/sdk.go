package sdk

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func makeHTTPError(code int) error {
	return fmt.Errorf("HTTP status = %d", code)
}

// Common error codes for HTTP responses
var (
	ErrorBadRequest          = makeHTTPError(http.StatusBadRequest)
	ErrorUnauthorized        = makeHTTPError(http.StatusUnauthorized)
	ErrorNotFound            = makeHTTPError(http.StatusNotFound)
	ErrorTooManyRequests     = makeHTTPError(http.StatusTooManyRequests)
	ErrorInternalServerError = makeHTTPError(http.StatusInternalServerError)
)

// StravaSDK wraps API calls to Strava
type StravaSDK interface {
	// authentication APIs
	ExchangeAuthToken(request *TokenExchangeCode) (*AuthorizationCodeResponse, error)
	RefreshAuthToken(refreshToken string) (*StravaTokens, error)

	// athlete APIs
	ListAllActivities(ctx context.Context, token string) ([]Activity, error)
	GetActivitiesByPage(ctx context.Context, token string, page int, perPage int) ([]Activity, error)
	GetActivityBytes(ctx context.Context, token string, activityID int64) ([]byte, error)
}

type StravaSDKConfig struct {
	Timeout      time.Duration
	ClientID     string
	ClientSecret string
}

// NewStravaSDK create a new SDK
func NewStravaSDK(config StravaSDKConfig) StravaSDK {
	return sdkImpl{
		client:       newHTTPClient(config.Timeout),
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
	}
}
