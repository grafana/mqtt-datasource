package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/toddtreece/mqtt-datasource/pkg/plugin"
)

func main() {
	err := datasource.Serve(plugin.GetDatasourceOpts())

	if err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
