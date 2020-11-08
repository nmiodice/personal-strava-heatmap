package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
)

type oauthClient struct {
	httpClient         *resty.Client
	stravaClientID     string
	stravaClientSecret string
}

type oauthDB struct {
	db *database.DB
}

type OAuthService struct {
	client oauthClient
	db     oauthDB
}

func NewOAuthService(httpClient *resty.Client, db *database.DB, stravaClientID, stravaClientSecret string) *OAuthService {
	return &OAuthService{
		client: oauthClient{
			httpClient:         httpClient,
			stravaClientID:     stravaClientID,
			stravaClientSecret: stravaClientSecret,
		},
		db: oauthDB{
			db: db,
		},
	}
}

func (o OAuthService) ExchangeAuthToken(ctx context.Context, request *TokenExchangeCode) (*AthleteToken, error) {

	authCodeResponse, err := o.client.doExchangeAuthToken(request)
	if err != nil {
		return nil, err
	}

	err = o.db.persistTokens(ctx, authCodeResponse.Athlete.ID, authCodeResponse.tokens())
	if err != nil {
		return nil, err
	}

	response := &AthleteToken{
		AccessToken: authCodeResponse.AccessToken,
		Athlete:     authCodeResponse.Athlete.ID,
	}

	return response, nil
}

func (o OAuthService) RefreshAuthToken(ctx context.Context, athleteID int) (*AthleteToken, error) {
	oldTokens, err := o.db.getTokensForAthlete(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	newTokens, err := o.client.doRefreshAuthToken(oldTokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	err = o.db.persistTokens(ctx, athleteID, newTokens)
	if err != nil {
		return nil, err
	}

	response := &AthleteToken{
		AccessToken: newTokens.AccessToken,
		Athlete:     athleteID,
	}

	return response, nil
}

func (c oauthClient) doRefreshAuthToken(refreshToken string) (*stravaTokens, error) {
	res, err := c.httpClient.R().
		SetFormData(map[string]string{
			"client_id":     c.stravaClientID,
			"client_secret": c.stravaClientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
		}).
		Post("https://www.strava.com/api/v3/oauth/token")

	if err != nil {
		return nil, err
	}

	tokens := &stravaTokens{}
	err = json.Unmarshal(res.Body(), tokens)
	return tokens, err
}

func (c oauthClient) doExchangeAuthToken(request *TokenExchangeCode) (*authorizationCodeResponse, error) {
	res, err := c.httpClient.R().
		SetFormData(map[string]string{
			"client_id":     c.stravaClientID,
			"client_secret": c.stravaClientSecret,
			"grant_type":    "authorization_code",
			"code":          request.Code,
		}).
		Post("https://www.strava.com/api/v3/oauth/token")

	if err != nil {
		return nil, err
	}

	authCodeResponse := &authorizationCodeResponse{}
	err = json.Unmarshal(res.Body(), authCodeResponse)
	return authCodeResponse, err
}

func (d oauthDB) persistTokens(ctx context.Context, athleteID int, tokens *stravaTokens) error {
	return d.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(
			ctx,
			insertTokensSQL,
			athleteID,
			tokens.AccessToken,
			time.Unix(tokens.ExpiresAt, 0),
			tokens.RefreshToken)

		var athleteID int
		if err := row.Scan(&athleteID); err != nil {
			return fmt.Errorf("fetching athlete_id after refresh token insert: %w", err)
		}

		return nil
	})
}

func (d oauthDB) getTokensForAthlete(ctx context.Context, athleteID int) (*stravaTokens, error) {
	tokens := stravaTokens{}
	expiresAt := time.Time{}

	err := d.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(
			ctx,
			getTokensSQL,
			athleteID)

		if err := row.Scan(&tokens.AccessToken, &expiresAt, &tokens.RefreshToken); err != nil {
			return fmt.Errorf("fetching refresh_token for athlete: %w, %d", err, athleteID)
		}

		return nil
	})
	tokens.ExpiresAt = expiresAt.UTC().Unix()
	return &tokens, err
}

func (d oauthDB) getAthleteForAuthToken(ctx context.Context, authToken string) (int, error) {
	var athleteID int

	err := d.db.InTx(ctx, pgx.Serializable, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, getAthleteForTokenSQL, authToken)
		if err := row.Scan(&athleteID); err != nil {
			return fmt.Errorf("fetching athlete for authToken: %w", err)
		}

		return nil
	})
	return athleteID, err
}

var insertTokensSQL = `
INSERT INTO
	StravaToken
	(athlete_id, access_token, access_token_expires_at, refresh_token)
VALUES
	($1, $2, $3, $4)
ON CONFLICT (athlete_id) 
	DO UPDATE
		SET athlete_id = $1, access_token = $2, access_token_expires_at = $3, refresh_token = $4
RETURNING
	athlete_id
`

var getTokensSQL = `
SELECT
	access_token, access_token_expires_at, refresh_token
FROM
	StravaToken
WHERE
	athlete_id = $1
`

var getAthleteForTokenSQL = `
SELECT
	athlete_id
FROM
	StravaToken
WHERE
	access_token = $1
`
