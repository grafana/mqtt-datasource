package main

import (
	"os"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/plugin"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/mqtt"
)

func main() {
	settings := &mqtt.Options{}
	settings.URI = "tls://a1ovt7grzmwsh8-ats.iot.us-east-1.amazonaws.com:8883"
	client, err := mqtt.NewClient(*settings)
	fmt.Println(client)
}
