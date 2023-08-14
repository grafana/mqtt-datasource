package main

import (
	_ "os"
	"fmt"

	_ "github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	_ "github.com/grafana/grafana-plugin-sdk-go/backend/log"
	_ "github.com/ISSACS-PSG/mqtt-datasource/pkg/plugin"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/mqtt"
)

func main() {
	settings := &mqtt.Options{}
	settings.URI = "tls://a1ovt7grzmwsh8-ats.iot.us-east-1.amazonaws.com:8883"
	client, _ := mqtt.NewClient(*settings)
	client.Subscribe("100ms/printers/status")
	fmt.Println(client)
	for {
		
	}
}
