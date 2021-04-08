package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/mitchellh/mapstructure"
	"github.com/toddtreece/mqtt-datasource/pkg/mqtt"
)

func LoadOptions(config backend.DataSourceInstanceSettings) (mqtt.Options, error) {
	options := mqtt.Options{}

	if err := json.Unmarshal(config.JSONData, &options); err != nil {
		return options, fmt.Errorf("could not unmarshal json: %w", err)
	}

	if config.DecryptedSecureJSONData == nil {
		return options, nil
	}

	if err := mapstructure.Decode(config.DecryptedSecureJSONData, &options); err != nil {
		return options, fmt.Errorf("could not decode secure options: %w", err)
	}

	return options, nil
}
