package main

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/mitchellh/mapstructure"
)

// Settings filled from json settings
type Settings struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	Username string `json:"username"`
	Password string `json:"-"`
}

// LoadSettings will read and validate Settings from the DataSourceConfg
func LoadSettings(config backend.DataSourceInstanceSettings) (Settings, error) {
	settings := Settings{}

	if err := json.Unmarshal(config.JSONData, &settings); err != nil {
		return settings, fmt.Errorf("could not unmarshal json: %w", err)
	}

	if config.DecryptedSecureJSONData == nil {
		return settings, nil
	}

	if err := mapstructure.Decode(config.DecryptedSecureJSONData, &settings); err != nil {
		return settings, fmt.Errorf("could not decode secure settings: %w", err)
	}

	return settings, nil
}
