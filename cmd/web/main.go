package main

import (
	"hackaton/internal/app"

	"go.uber.org/fx"
)

//	@title			Hackaton API
//	@version		0.1
//	@description	REST API хакатон-проекта.

//	@BasePath	/v1

func main() {
	fx.New(
		app.ModuleBase(),
		app.ModuleDB(),
		app.ModuleRepositories(),
		app.ModuleServices(),
		app.ModuleWebServer(),
		app.ModuleV1Handlers(),
		app.ModuleSwagger(),
		app.ModuleRunWebServer(),
	).Run()
}
