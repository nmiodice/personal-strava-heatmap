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
)

func configureRouter(config *backend.Config, routes *backend.HttpRoutes) *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob(config.TemplatePath + "/*")

	router.GET("/", routes.IndexRoute)
	router.GET("/index.html", routes.IndexRoute)
	router.GET("/map.html", routes.MapRoute)
	router.GET("/tokenexchange", routes.TokenExchange)
	router.GET("/profile", routes.ProfileRoute)
	router.GET("/unprocessedactivities", routes.UnprocessedActivitiesRoute)
	router.GET("/syncactivities", routes.SyncActivitiesRoute)
	router.GET("/buildmap", routes.BuildMapRoute)

	router.Use(routes.StaticFileServer("/static"))

	return router
}

func main() {
	ctx := context.Background()
	config := backend.GetConfig(ctx)
	deps, err := backend.GetDependencies(ctx, config)
	if err != nil {
		log.Fatalf("Error configuring application dependencies: %+v", err)
	}

	routes := backend.GetRoutes(config, deps)
	router := configureRouter(config, routes)

	router.Run(fmt.Sprintf(":%d", config.HttpServer.Port))
}
