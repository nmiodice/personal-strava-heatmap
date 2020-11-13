// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package is the primary infected keys upload service.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nmiodice/personal-strava-heatmap/internal/backend"
	"github.com/nmiodice/personal-strava-heatmap/internal/background/athlete"
	"github.com/nmiodice/personal-strava-heatmap/internal/background/processor"
)

const (
	tokenRefreshLockID        = 1
	activityListRefreshLockID = 2
	activityDownloadLockID    = 3
)

func configureRouter(config *backend.Config, routes *backend.HttpRoutes) *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob(config.TemplatePath + "/*")

	router.GET("/", routes.IndexRoute)
	router.GET("/index.html", routes.IndexRoute)
	router.GET("/map.html", routes.MapRoute)
	router.GET("/tokenexchange", routes.TokenExchange)

	router.Use(routes.StaticFileServer("/static"))

	return router
}

func runHTTPServerForever(config *backend.Config, deps *backend.Dependencies) {
	routes := backend.GetRoutes(config, deps)
	router := configureRouter(config, routes)

	router.Run(fmt.Sprintf(":%d", config.HttpServer.Port))
}

func triggerBackgroundJobs(ctx context.Context, config *backend.Config, deps *backend.Dependencies) {
	// refresh access tokens
	processor.RunForever(ctx, athlete.AthleteTokenRefreshConfig(
		ctx,
		deps.Strava,
		deps.MakeLockFunc(tokenRefreshLockID)))

	// refresh activities for athletes
	processor.RunForever(ctx, athlete.AthleteActivityListRefreshConfig(
		ctx,
		deps.Strava,
		deps.MakeLockFunc(activityListRefreshLockID)))

	// sync missing ride data for athletes
	processor.RunForever(ctx, athlete.AthleteActivityStreamRefreshConfig(
		ctx,
		deps.Strava,
		deps.Map,
		deps.MakeLockFunc(activityDownloadLockID)))
}

func main() {
	ctx := context.Background()
	config := backend.GetConfig(ctx)
	deps, err := backend.GetDependencies(ctx, config)
	if err != nil {
		log.Fatalf("Error configuring application dependencies: %+v", err)
	}

	triggerBackgroundJobs(ctx, config, deps)
	runHTTPServerForever(config, deps)
}
