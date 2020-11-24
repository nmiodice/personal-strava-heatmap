package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	resty "github.com/go-resty/resty/v2"
)

type sdkImpl struct {
	client       *resty.Client
	clientID     string
	clientSecret string
}

const (
	// according to https://developers.strava.com/docs/
	maxPaginatedResults = 200
	apiRootURL          = "https://www.strava.com/api/v3/"
)

func (sdk sdkImpl) ExchangeAuthToken(request *TokenExchangeCode) (*AuthorizationCodeResponse, error) {
	res, err := sdk.client.R().
		SetFormData(map[string]string{
			"client_id":     sdk.clientID,
			"client_secret": sdk.clientSecret,
			"grant_type":    "authorization_code",
			"code":          request.Code,
		}).
		Post(apiRootURL + "oauth/token")

	if err != nil {
		return nil, err
	}

	authCodeResponse := &AuthorizationCodeResponse{}
	err = json.Unmarshal(res.Body(), authCodeResponse)
	return authCodeResponse, err
}

func (sdk sdkImpl) RefreshAuthToken(refreshToken string) (*StravaTokens, error) {
	res, err := sdk.client.R().
		SetFormData(map[string]string{
			"client_id":     sdk.clientID,
			"client_secret": sdk.clientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
		}).
		Post(apiRootURL + "oauth/token")

	if err != nil {
		return nil, err
	}

	tokens := &StravaTokens{}
	err = json.Unmarshal(res.Body(), tokens)
	return tokens, err
}

func (sdk sdkImpl) ListAllActivities(ctx context.Context, token string) ([]Activity, error) {
	page := 1
	activities := []Activity{}
	for {
		pageActivities, err := sdk.GetActivitiesByPage(ctx, token, page, maxPaginatedResults)
		if err != nil {
			return activities, err
		}

		if len(pageActivities) == 0 {
			return activities, nil
		}

		activities = append(activities, pageActivities...)
		page++
	}
}

// GetActivitiesByPage return the activities on a particular page
func (sdk sdkImpl) GetActivitiesByPage(ctx context.Context, token string, page int, perPage int) ([]Activity, error) {
	activities := []Activity{}

	res, err := sdk.client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParams(map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
		}).
		Get(apiRootURL + "activities")

	if err != nil {
		return activities, err
	}

	err = json.Unmarshal(res.Body(), &activities)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// GetActivityBytes return raw representation of activities
func (sdk sdkImpl) GetActivityBytes(ctx context.Context, token string, activityID int64) ([]byte, error) {
	url := fmt.Sprintf(apiRootURL+"activities/%d/streams?keys=latlng", activityID)
	res, err := sdk.client.R().
		SetHeader("Authorization", "Bearer "+token).
		Get(url)

	if err != nil {
		return nil, err
	}
	return res.Body(), nil
}
