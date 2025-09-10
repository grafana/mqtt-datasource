# Changelog

## [1.2.0] - 2025-09-10

- Add support for raw string values
- Add streaming key to the requests
- Use go 1.25 for building the plugin
- Implement error source and use context logger

## [1.1.0] - 2025-07-28

- Fix session reuse for subscriptions after reconnect
- Add client ID setting for MQTT connections
- Use URL-safe Base64 encoding for topic names
- Fix clear field data functionality
- Upgrade dependencies (grafana-plugin-sdk-go, form-data, @babel/runtime)
- Introduce changesets for better release management
- Update repository links and workflows
- Stable release (removed beta status)

## [1.1.0-beta.3] - 2025-02-25

- Upgrade dependencies

## [1.1.0-beta.2] - 2024-08-21

- Upgrade dependencies

## [1.1.0-beta.1] - 2024-06-06

- Add support for TLS Client Authentication
- Add TLS Skip Verify option
- Add Support for specifying a custom CA Certificate

## [1.0.0-beta.4] - 2024-03-21

- Add support for MQTT Wildcards

## [1.0.0-beta.3] - 2023-08-17

- Fix for #44

## [1.0.0-beta.2] - 2023-04-25

- Update Plugin SDK dependency

## [1.0.0-beta.1] - 2022-12-01

- Initial release
