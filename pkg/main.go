package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
)

func main() {

	im := datasource.NewInstanceManager(plugin.NewServerInstance)
	err := datasource.Serve(plugin.GetDatasourceOpts(im))

	if err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
