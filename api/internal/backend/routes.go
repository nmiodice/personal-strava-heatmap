package backend

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/nmiodice/personal-strava-heatmap/internal/batch"
	"github.com/nmiodice/personal-strava-heatmap/internal/concurrency"
	"github.com/nmiodice/personal-strava-heatmap/internal/maps"
	"github.com/nmiodice/personal-strava-heatmap/internal/queue"
	"github.com/nmiodice/personal-strava-heatmap/internal/storage"
	"github.com/nmiodice/personal-strava-heatmap/internal/strava"
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
	IndexRoute                 gin.HandlerFunc
	MapRoute                   gin.HandlerFunc
	ProfileRoute               gin.HandlerFunc
	TokenExchange              gin.HandlerFunc
	UnprocessedActivitiesRoute gin.HandlerFunc
	SyncActivitiesRoute        gin.HandlerFunc
	BuildMapRoute              gin.HandlerFunc
	StaticFileServer           func(string) gin.HandlerFunc
}

func GetRoutes(config *Config, deps *Dependencies) *HttpRoutes {
	return &HttpRoutes{
		TokenExchange:              getTokenExchangeRouteFunc(deps.Strava),
		UnprocessedActivitiesRoute: getUnprocessedActivitiesRoute(deps.Strava),
		SyncActivitiesRoute:        getSyncActivitiesRoute(deps.Strava),
		BuildMapRoute: getBuildMapRoute(
			config.Storage.ConcurrencyLimit,
			config.Map,
			deps.Strava,
			deps.Storage,
			deps.Map,
			deps.Queue,
			config.Queue),
		ProfileRoute: templateFileRoute("profile.html", gin.H{}),
		IndexRoute: templateFileRoute("index.html", gin.H{
			"title":            "Personalized Strava Heatmap",
			"strava_client_id": config.Strava.ClientID,
		}),
		MapRoute: getMapRoute(
			"map.html",
			deps.Strava,
			config.Storage,
			config.Map),
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

var getMapRoute = func(templateFileName string, stravaSvc *strava.StravaService, storageConfig StorageConfig, mapConfig MapConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query(QueryParamToken)
		if token == "" {
			c.JSON(401, gin.H{
				ResponseError: "Missing Token",
			})
			return
		}

		mapID, err := stravaSvc.Athlete.GetOrCreateMapID(c.Request.Context(), token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		c.HTML(http.StatusOK, templateFileName, gin.H{
			"map_id":        mapID,
			"map_api_key":   mapConfig.MapsAPIKey,
			"tile_endpoint": fmt.Sprintf("https://%s.blob.core.windows.net/%s/", storageConfig.AccountName, storageConfig.UploadContainerName),
		})
	}
}

var getTokenExchangeRouteFunc = func(stravaSvc *strava.StravaService) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := stravaSvc.Auth.ExchangeAuthToken(c.Request.Context(), &sdk.TokenExchangeCode{
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
		c.Redirect(301, fmt.Sprintf("/profile/?%s", params.Encode()))
	}
}

var getUnprocessedActivitiesRoute = func(stravaSvc *strava.StravaService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query(QueryParamToken)
		if token == "" {
			c.JSON(401, gin.H{
				ResponseError: "Missing Token",
			})
			return
		}

		res, err := stravaSvc.Athlete.RefreshActivities(c.Request.Context(), token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError: err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			ResponseActivityRefresh: res,
		})
	}
}

var getSyncActivitiesRoute = func(stravaSvc *strava.StravaService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query(QueryParamToken)
		if token == "" {
			c.JSON(401, gin.H{
				ResponseError: "Missing Token",
			})
			return
		}

		res, err := stravaSvc.Athlete.SyncActivities(c.Request.Context(), token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError:            err.Error(),
				ResponseActivitiesSynced: res,
			})
			return
		}

		c.JSON(200, gin.H{
			ResponseActivitiesSynced: res,
		})
	}
}

var getBuildMapRoute = func(
	storageConcurrencyLimit int,
	mapConfig MapConfig,
	stravaSvc *strava.StravaService,
	storageSvc *storage.AzureBlobstore,
	mapSvc *maps.MapService,
	queueSvc queue.QueueService,
	queueConfig QueueConfig,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		token := c.Query(QueryParamToken)
		if token == "" {
			c.JSON(401, gin.H{
				ResponseError: "Missing Token",
			})
			return
		}

		dataRefs, err := stravaSvc.Athlete.GetActivityDataRefs(ctx, token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError:  err.Error(),
				ResponseStatus: "failed",
			})
			return
		}

		mapSem := concurrency.NewSemaphore(1)
		tiles := maps.NewTileSet()

		funcs := [](func() error){}
		for _, ref := range dataRefs {
			theRef := ref
			funcs = append(funcs, func() error {
				bytes, err := storageSvc.GetObjectBytes(ctx, theRef)
				if err != nil {
					return err
				}

				mapSem.Acquire(1)
				defer mapSem.Release(1)

				mapSvc.AddToTileSet(bytes, mapConfig.MinTileZoom, mapConfig.MaxTileZoom, &tiles)
				return nil
			})
		}

		if err = concurrency.NewSemaphore(storageConcurrencyLimit).WithRateLimit(funcs, true); err != nil {
			c.JSON(500, gin.H{
				ResponseError:  err.Error(),
				ResponseStatus: "failed",
			})
			return
		}

		mapParams := mapSvc.ComputeMapParams(&tiles)
		messages := make([]interface{}, len(mapParams))
		for idx, p := range mapParams {
			messages[idx] = p
		}

		athleteID, err := stravaSvc.Athlete.GetAthleteForAuthToken(ctx, token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError:  err.Error(),
				ResponseStatus: "failed",
			})
			return
		}

		mapID, err := stravaSvc.Athlete.GetOrCreateMapID(ctx, token)
		if err != nil {
			c.JSON(500, gin.H{
				ResponseError:  err.Error(),
				ResponseStatus: "failed",
			})
			return
		}
		messageBatches := batch.ToBatchesWithTransformer(messages, queueConfig.BatchSize, func(batch []interface{}) interface{} {
			return map[string]interface{}{
				"coords":     batch,
				"athlete_id": athleteID,
				"map_id":     mapID,
			}
		})

		if err = queueSvc.Enqueue(ctx, messageBatches...); err != nil {
			c.JSON(500, gin.H{
				ResponseError:  err.Error(),
				ResponseStatus: "failed",
			})
			return
		}

		c.JSON(202, gin.H{
			ResponseStatus:          "started",
			ResponseTileBatchCount:  len(messageBatches),
			ResponseActivitiesCount: len(dataRefs),
		})
	}
}
