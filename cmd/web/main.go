package main

import (
	"hackaton/internal/app"

	"go.uber.org/fx"
)

//	@title			Граф денег — API
//	@version		0.1
//	@description	Роли, кластеры и приоритеты узлов транзакционной сети (HackAlem AI).

//	@BasePath	/v1

func main() {
	fx.New(
		app.ModuleBase(),
		app.ModuleRepositories(),
		app.ModuleServices(),
		app.ModuleWebServer(),
		app.ModuleV1Handlers(),
		app.ModuleSwagger(),
		app.ModuleRunWebServer(),
	).Run()
}
