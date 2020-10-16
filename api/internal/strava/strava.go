package strava

type StravaService struct {
	Athlete *AthleteService
	Auth    *OAuthService
}
