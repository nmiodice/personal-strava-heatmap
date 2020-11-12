package strava

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/nmiodice/personal-strava-heatmap/internal/database"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

type oauthDB struct {
	db *database.DB
}

type OAuthService struct {
	stravaSDK sdk.StravaSDK
	db        oauthDB
}

func NewOAuthService(stravaSDK sdk.StravaSDK, db *database.DB) *OAuthService {
	return &OAuthService{
		stravaSDK: stravaSDK,
		db: oauthDB{
			db: db,
		},
	}
}

func (o OAuthService) ExchangeAuthToken(ctx context.Context, request *sdk.TokenExchangeCode) (*sdk.AthleteToken, error) {

	authCodeResponse, err := o.stravaSDK.ExchangeAuthToken(request)
	if err != nil {
		return nil, err
	}

	err = o.db.persistTokens(ctx, authCodeResponse.Athlete.ID, authCodeResponse.Tokens())
	if err != nil {
		return nil, err
	}

	response := &sdk.AthleteToken{
		AccessToken: authCodeResponse.AccessToken,
		Athlete:     authCodeResponse.Athlete.ID,
	}

	return response, nil
}

func (o OAuthService) RefreshAuthToken(ctx context.Context, athleteID int) (*sdk.AthleteToken, error) {
	oldTokens, err := o.db.getTokensForAthlete(ctx, athleteID)
	if err != nil {
		return nil, err
	}

	newTokens, err := o.stravaSDK.RefreshAuthToken(oldTokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	err = o.db.persistTokens(ctx, athleteID, newTokens)
	if err != nil {
		return nil, err
	}

	response := &sdk.AthleteToken{
		AccessToken: newTokens.AccessToken,
		Athlete:     athleteID,
	}

	return response, nil
}

func (d oauthDB) persistTokens(ctx context.Context, athleteID int, tokens *sdk.StravaTokens) error {
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

func (d oauthDB) getTokensForAthlete(ctx context.Context, athleteID int) (*sdk.StravaTokens, error) {
	tokens := sdk.StravaTokens{}
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
