package sdk

import "time"

type AuthorizationCodeResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	Athlete      struct {
		ID            int         `json:"id"`
		Username      string      `json:"username"`
		ResourceState int         `json:"resource_state"`
		Firstname     string      `json:"firstname"`
		Lastname      string      `json:"lastname"`
		City          interface{} `json:"city"`
		State         interface{} `json:"state"`
		Country       interface{} `json:"country"`
		Sex           string      `json:"sex"`
		Premium       bool        `json:"premium"`
		Summit        bool        `json:"summit"`
		CreatedAt     time.Time   `json:"created_at"`
		UpdatedAt     time.Time   `json:"updated_at"`
		BadgeTypeID   int         `json:"badge_type_id"`
		ProfileMedium string      `json:"profile_medium"`
		Profile       string      `json:"profile"`
		Friend        interface{} `json:"friend"`
		Follower      interface{} `json:"follower"`
	} `json:"athlete"`
}

func (acr AuthorizationCodeResponse) Tokens() *StravaTokens {
	return &StravaTokens{
		AccessToken:  acr.AccessToken,
		ExpiresAt:    acr.ExpiresAt,
		RefreshToken: acr.RefreshToken,
	}
}

type StravaTokens struct {
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
}

type TokenExchangeCode struct {
	Code string
}

type AthleteToken struct {
	AccessToken string
	Athlete     int
}

type Activities struct {
	Collection []Activity
}

type Activity struct {
	Athlete struct {
		ID int `json:"id"`
	} `json:"athlete"`
	ID int64 `json:"id"`
}
