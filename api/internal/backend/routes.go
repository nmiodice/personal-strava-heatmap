package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/nmiodice/personal-strava-heatmap/internal/orchestrator"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava/sdk"
)

const (
	ResponseError              = "Error"
	ResponseActivityRefresh    = "ActivityRefresh"
	ResponseActivitiesSynced   = "ActivitiesSynced"
	QueryParamCode             = "code"
	QueryParamToken            = "token"
	ResponseStatus             = "status"
	ResponseActivitiesIncluded = "activities"
	ResponseActivitiesCount    = "activity_count"
	ResponseTileBatchCount     = "tile_batch_count"
	WebsiteName                = "Personal Heatmap"
)

type HttpRoutes struct {
	IndexRoute              gin.HandlerFunc
	MapRoute                gin.HandlerFunc
	MapProcessingStateRoute gin.HandlerFunc
	TokenExchange           gin.HandlerFunc
	LogoutRoute             gin.HandlerFunc
	StaticFileServer        func(string) gin.HandlerFunc
}

func GetRoutes(config *Config, deps *Dependencies) *HttpRoutes {
	return &HttpRoutes{
		TokenExchange:           getTokenExchangeRouteFunc(config, deps),
		IndexRoute:              getIndexRoute("index.html", config, deps),
		MapRoute:                getMapRoute("map.html", config, deps),
		LogoutRoute:             getLogoutRoute(),
		MapProcessingStateRoute: getMapProcessingStateRoute(config, deps),
		StaticFileServer: func(urlPrefix string) gin.HandlerFunc {
			return static.Serve(urlPrefix, static.LocalFile(config.StaticFileRoot, false))
		},
	}
}

func getMapProcessingStateRoute(config *Config, deps *Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil || token == "" {
			c.Redirect(301, "/")
			return
		}

		ctx := c.Request.Context()
		mapProcessingState, err := deps.Map.GetProcessingStateForAthlete(ctx, token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		athleteID, err := deps.Strava.Athlete.GetAthleteForAuthToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		state, err := deps.State.GetState(ctx, athleteID)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"athlete_state": gin.H{
				"state": state,
			},
			"map_state": gin.H{
				"processing": mapProcessingState.Queued,
				"completed":  mapProcessingState.Complete,
				"failed":     mapProcessingState.Failed,
			},
		})
		return
	}
}

func getMapRoute(templateFileName string, config *Config, deps *Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil || token == "" {
			c.Redirect(301, "/")
			return
		}

		mapID, err := deps.Strava.Athlete.GetOrCreateMapID(c.Request.Context(), token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		c.HTML(http.StatusOK, templateFileName, gin.H{
			"title":         WebsiteName,
			"map_id":        mapID,
			"map_api_key":   config.Map.MapsAPIKey,
			"tile_endpoint": fmt.Sprintf("https://%s.blob.core.windows.net/%s/", config.Storage.AccountName, config.Storage.UploadContainerName),
		})
	}
}

func getIndexRoute(templateFileName string, config *Config, deps *Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err == nil && token != "" {
			c.Redirect(301, "/map.html")
			return
		}

		// helps with login/logout
		c.Header("Cache-Control", "no-cache")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":            WebsiteName,
			"strava_client_id": config.Strava.ClientID,
		})
	}
}

func getLogoutRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.SetCookie("token", "", -1, "", "", false, true)
		c.Redirect(301, "/index.html")
		return
	}
}

func getTokenExchangeRouteFunc(config *Config, deps *Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := deps.Strava.Auth.ExchangeAuthToken(c.Request.Context(), &sdk.TokenExchangeCode{
			Code: c.Query(QueryParamCode),
		})

		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		c.SetCookie("token", res.AccessToken, 0, "", "", false, true)
		c.Redirect(301, "/map.html/")

		// kick off background job to update profile and rebuild map
		go func() {
			orchestrator.UpdateAthleteMap(
				deps.Strava,
				deps.Map,
				deps.State,
				res.Athlete,
				res.AccessToken,
				context.Background())
		}()
	}
}
