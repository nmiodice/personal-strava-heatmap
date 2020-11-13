package backend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
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
)

type HttpRoutes struct {
	IndexRoute       gin.HandlerFunc
	MapRoute         gin.HandlerFunc
	TokenExchange    gin.HandlerFunc
	StaticFileServer func(string) gin.HandlerFunc
}

func GetRoutes(config *Config, deps *Dependencies) *HttpRoutes {
	return &HttpRoutes{
		TokenExchange: getTokenExchangeRouteFunc(config, deps),
		MapRoute:      getMapRoute("map.html", config, deps),
		IndexRoute: templateFileRoute("index.html", gin.H{
			"title":            "Personalized Strava Heatmap",
			"strava_client_id": config.Strava.ClientID,
		}),
		StaticFileServer: func(urlPrefix string) gin.HandlerFunc {
			return static.Serve(urlPrefix, static.LocalFile(config.StaticFileRoot, false))
		},
	}
}

func templateFileRoute(templateFileName string, params gin.H) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, templateFileName, params)
	}
}

var getMapRoute = func(templateFileName string, config *Config, deps *Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query(QueryParamToken)
		if token == "" {
			c.JSON(401, gin.H{
				ResponseError: "Missing Token",
			})
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
			"map_id":        mapID,
			"map_api_key":   config.Map.MapsAPIKey,
			"tile_endpoint": fmt.Sprintf("https://%s.blob.core.windows.net/%s/", config.Storage.AccountName, config.Storage.UploadContainerName),
		})
	}
}

var getTokenExchangeRouteFunc = func(config *Config, deps *Dependencies) gin.HandlerFunc {
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

		params := url.Values{}
		params.Add(QueryParamToken, res.AccessToken)
		c.Redirect(301, fmt.Sprintf("/map.html/?%s", params.Encode()))

		// kick off background job to update profile and rebuild map
		go func() {
			bgCtx := context.Background()

			log.Printf("importing new activities for athlete '%d'", res.Athlete)
			_, err := deps.Strava.Athlete.ImportNewActivities(bgCtx, res.AccessToken)
			if err != nil {
				log.Printf("error encountered importing new activities for athlete '%d': %+v", res.Athlete, err)
			}

			log.Printf("importing new activity streams for athlete '%d'", res.Athlete)
			imported, err := deps.Strava.Athlete.ImportMissingActivityStreams(bgCtx, res.AccessToken)
			if err != nil {
				log.Printf("error encountered importing new activity streams for athlete '%d': %+v", res.Athlete, err)
			}

			if imported > 0 {
				log.Printf("rebuilding map for athlete '%d'", res.Athlete)
				dataRefs, messageBatches, err := deps.Map.RebuildMapForAthlete(bgCtx, res.AccessToken)
				if err != nil {
					log.Printf("error encountered rebuilding map for athlete '%d': %+v", res.Athlete, err)
					return
				}
				log.Printf("rebuilt map using '%d' data refs and '%d' queued messages for athlete '%d'", len(dataRefs), len(messageBatches), res.Athlete)
			}
		}()
	}
}
