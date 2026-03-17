---
aliases:
  - /docs/plugins/grafana-mqtt-datasource/experimental/
description: Experimental features of the MQTT Datasource
keywords:
  - grafana
  - mqtt
  - topics
  - streaming
  - json
  - experimental
  - publish
labels:
  products:
    - cloud
    - enterprise
    - oss
menuTitle: Experimental Features
title: Experimental Features
weight: 300
review_date: 2026-03-17
---

# Experimental Features

## Publishing

The MQTT data source includes experimental support for publishing messages to MQTT topics. This feature allows you to send data to MQTT brokers directly from Grafana dashboards, panels, or through programmatic queries.

### Enable Publishing

#### Via GUI Configuration

1. Navigate to **Configuration** → **Data sources** → **MQTT**
2. Scroll to the **Experimental** section
3. Toggle **Enable Publishing** to `On`
4. Set the **Publishing Timeout** if needed (default: `1s`, examples: `500ms`, `2s`)

#### Via Provisioning

Add the publishing configuration to your datasource YAML:

```yaml
apiVersion: 1

datasources:
  - name: MQTT
    type: grafana-mqtt-datasource
    jsonData:
      # ...
      enablePublishing: true
      publishingTimeout: "1s"
```

### Query Structure for Publishing

To publish a message, include a `payload` object in your query. The presence of a non-empty payload triggers publishing mode instead of subscription.

#### Basic Publishing Query

```json
{
  "topic": "home/device/command",
  "payload": {
    "action": "turn_on",
    "brightness": 75
  }
}
```

#### Publishing with Response Topic

For request-response patterns, specify a response topic to capture the reply:

```json
{
  "topic": "home/device/command", 
  "payload": {
    "action": "get_status"
  },
  "response": "home/device/response"
}
```

The data source will:

1. Subscribe to the response topic
2. Publish the payload to the command topic  
3. Wait for a response (up to the configured timeout)
4. Return the response data in the query result

#### Response Frame Structure

When a response is received, the data source returns a frame with the following structure:

```json
{
  "frames": [
    {
      "name": "Response",
      "fields": [
        {
          "name": "Body",
          "type": "other",
          "values": ["<raw_response_payload>"]
        }
      ]
    }
  ]
}
```

The response payload is returned as a raw JSON message in the `Body` field, preserving the exact format sent by the MQTT broker. For example, if the device responds with `{"status": "on", "temperature": 22.5}`, this JSON will be available in the `Body` field.

### Using Publishing from Panel Plugins

Panel plugins can send publishing queries using Grafana's standard data source API. Access the data source through the `getDataSourceSrv()` method:

```typescript
import { getDataSourceSrv } from '@grafana/runtime';

// Get the data source instance
const dataSourceSrv = getDataSourceSrv();
const mqttDataSource = await dataSourceSrv.get('MQTT'); // Replace with your datasource name

// Create a publishing query  
const publishQuery = {
  refId: 'A',
  topic: 'home/thermostat/set',
  payload: {
    temperature: 22,
    mode: 'heat'
  },
  response: 'home/thermostat/status'
};

// Execute the query
const response = await mqttDataSource.query({
  targets: [publishQuery],
  range: {} // Required but not used for publishing
});
```

### Using Publishing via HTTP API

Since the MQTT data source extends `DataSourceWithBackend`, you can send publishing queries through Grafana's query API:

```bash
curl -X POST \
  http://grafana-url/api/ds/query \
  -H 'Authorization: Bearer <API_TOKEN>' \
  -H 'Content-Type: application/json' \
  -d '{
    "queries": [{
      "refId": "A", 
      "datasource": {"uid": "<DATASOURCE_UID>"},
      "topic": "sensors/temperature/set",
      "payload": {
        "value": 25.5,
        "unit": "celsius"
      }
    }]
  }'
```

### Limitations and Considerations

- Publishing is **experimental** and may change in future versions
- Queries with empty `payload` objects will default to subscription mode
- Response topics are optional but recommended for command-response patterns  
- Publishing requires appropriate MQTT broker permissions for the configured user
- Timeouts apply to both the publish operation and waiting for responses
- All payloads are JSON-encoded before publishing
