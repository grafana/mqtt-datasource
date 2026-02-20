---
aliases:
  - /docs/plugins/grafana-mqtt-datasource/query-editor/
description: Use the MQTT query editor to subscribe to topics and visualize streaming data in Grafana.
keywords:
  - grafana
  - mqtt
  - query editor
  - topics
  - streaming
  - json
labels:
  products:
    - cloud
    - oss
menuTitle: Query editor
title: MQTT query editor
weight: 300
last_reviewed: 2026-02-20
---

# MQTT query editor

This document explains how to use the MQTT query editor to subscribe to topics and visualize streaming data in your Grafana panels.

## Before you begin

- Ensure you have [configured the MQTT data source](https://grafana.com/docs/plugins/grafana-mqtt-datasource/latest/configure/).
- Verify the connection is working by checking that **Save & test** shows **MQTT Connected**.

## Key concepts

If you're new to MQTT, here are key terms used in this documentation.

| Term | Description |
|------|-------------|
| **Topic** | A UTF-8 string that the broker uses to route messages to subscribers. Topics are hierarchical, separated by `/` (for example, `home/bedroom/temperature`). |
| **Wildcard** | A special character in a topic filter that matches one or more topic levels. `+` matches a single level, `#` matches all remaining levels. |
| **QoS** | Quality of Service. This plugin uses QoS 0 (at most once delivery), meaning messages are delivered with best effort and aren't acknowledged. |
| **Retained message** | A message the broker stores and delivers to new subscribers immediately upon subscription. The plugin can receive retained messages on connect. |

## Create a query

To subscribe to an MQTT topic:

1. Select the **MQTT** data source.
1. In the **Topic** field, enter the MQTT topic you want to subscribe to (for example, `home/bedroom/temperature`).
1. The panel begins streaming data as soon as the topic is set.

Each query subscribes to a single topic. To subscribe to multiple topics, add additional queries to the panel.

## Topic wildcards

The MQTT data source supports standard MQTT wildcard characters in topic filters.

| Wildcard | Description | Example |
|----------|-------------|---------|
| `+` | Matches exactly one topic level. | `home/+/temperature` matches `home/bedroom/temperature` and `home/kitchen/temperature`, but not `home/bedroom/sensor/temperature`. |
| `#` | Matches all remaining topic levels. Must be the last character. | `home/#` matches `home/bedroom/temperature`, `home/kitchen/humidity`, and any other topic under `home/`. |

For the full specification on topic names and filters, refer to the [MQTT v3.1.1 specification](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc398718106).

## Supported data types

The plugin automatically detects the data type of each incoming message and creates appropriate data frame fields. The following types are supported.

| Data type | Example payload | Field type |
|-----------|-----------------|------------|
| **Number** | `23.5` | Float64 |
| **String** | `"online"` | String |
| **Boolean** | `true` | Bool |
| **JSON object** | `{"temperature": 23.5, "humidity": 60}` | One field per top-level key |
| **JSON array** | `[1, 2, 3]` | JSON |

When the plugin receives a JSON object, it extracts each top-level key into a separate field. For example, a message with `{"temperature": 23.5, "humidity": 60}` creates two fields: `temperature` (Float64) and `humidity` (Float64).

## Work with JSON data

For flat JSON objects, each top-level key becomes its own field automatically. Nested objects and arrays are stored as JSON-typed fields.

To extract values from nested JSON structures, use the **Extract fields** transformation:

1. Open the panel editor.
1. Click the **Transform data** tab.
1. Select **Extract fields**.
1. Choose the JSON field you want to extract values from.

For more information, refer to [Transform data](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/panels-visualizations/query-transform-data/transform-data/).

## Understand timestamps

The plugin attaches a timestamp to each message when it arrives at the Grafana server. These timestamps reflect when Grafana received the message, not when the event occurred at the source.

If your message payloads include a timestamp field, use the **Convert field type** transformation to parse it:

1. Open the panel editor.
1. Click the **Transform data** tab.
1. Select **Convert field type**.
1. Choose the field that contains the timestamp.
1. Set the target type to **Time**.

For more information, refer to [Convert field type](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/panels-visualizations/query-transform-data/transform-data/#convert-field-type).

## Streaming behavior

The MQTT data source uses the [Grafana Live](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/setup-grafana/set-up-grafana-live/) streaming API to push data to panels in real time.

The data flow works as follows:

1. When a panel with an MQTT query is opened, Grafana subscribes to the specified MQTT topic on the broker.
1. Incoming messages accumulate in a buffer on the backend.
1. At each query interval, the buffered messages are converted into a data frame and pushed to the panel.
1. When the panel is closed or the query is removed, Grafana unsubscribes from the topic.

{{< admonition type="note" >}}
Only data received while the panel is open is displayed. There's no historical backfill when you open a panel or change the time range. The dashboard time range doesn't affect which messages are shown.
{{< /admonition >}}

If multiple panels subscribe to the same topic, the plugin shares a single MQTT subscription and routes the data to each panel independently.
