---
aliases:
  - /docs/plugins/grafana-mqtt-datasource/configure/
description: Configure the MQTT data source in Grafana to connect to your MQTT broker.
keywords:
  - grafana
  - mqtt
  - configure
  - authentication
  - tls
  - provisioning
labels:
  products:
    - cloud
    - enterprise
    - oss
menuTitle: Configure
title: Configure the MQTT data source
weight: 200
review_date: 2026-02-20
---

# Configure the MQTT data source

This document explains how to configure the MQTT data source to connect to your MQTT broker.

## Before you begin

Before you configure the MQTT data source, ensure you have:

- **Grafana permissions:** Organization administrator role.
- **MQTT broker access:** The URI of a running MQTT v3.1.x broker that's reachable from the Grafana server.
- **Credentials:** Username and password, or TLS certificates, if your broker requires authentication.

## Add the data source

To add the MQTT data source to Grafana:

1. Click **Connections** in the left-side menu.
1. Click **Add new connection**.
1. Type `MQTT` in the search bar.
1. Select **MQTT**.
1. Click **Add new data source**.

## Connection settings

Use the following settings to configure the connection to your MQTT broker.

| Setting | Description |
|---------|-------------|
| **Name** | A display name for this data source instance. |
| **URI** | The URI of your MQTT broker. Include the scheme and port. Supported schemes: `tcp://` (unencrypted, default port `1883`), `tls://` (TLS-encrypted, default port `8883`), `ws://` (WebSocket, default port `80`), and `wss://` (WebSocket Secure, default port `443`). The aliases `mqtt://` (same as `tcp://`), `ssl://`, `tcps://`, and `mqtts://` (same as `tls://`) are also accepted. For example, `tcp://localhost:1883` or `tls://broker.example.com:8883`. If you omit the port, the default for the scheme is used. |
| **Client ID** | An optional MQTT client identifier. If left empty, Grafana generates a random ID in the format `grafana_<number>`. |

## Authentication

If your broker requires credentials, configure them in the **Authentication** section.

| Setting | Description |
|---------|-------------|
| **Username** | The username for MQTT broker authentication. |
| **Password** | The password for MQTT broker authentication. Stored securely in Grafana. |

## TLS authentication

The MQTT data source supports TLS for encrypted connections and mutual TLS (mTLS) for client certificate authentication.

| Setting | Description |
|---------|-------------|
| **Use TLS Client Auth** | Enable to authenticate with a client certificate and private key. When enabled, the **TLS Configuration** section appears. |
| **Skip TLS Verification** | Enable to skip verification of the broker's TLS certificate chain and host name. Use this only for testing or when connecting to brokers with self-signed certificates. |
| **With CA Cert** | Enable to provide a custom CA certificate for verifying the broker's server certificate. When enabled, the **TLS Configuration** section appears. |

When you enable **Use TLS Client Auth** or **With CA Cert**, configure the certificates in the **TLS Configuration** section that appears.

| Setting | Description |
|---------|-------------|
| **TLS CA Certificate** | The PEM-encoded CA certificate used to verify the broker's server certificate. Required when **With CA Cert** is enabled. |
| **TLS Client Certificate** | The PEM-encoded client certificate for mTLS authentication. Required when **Use TLS Client Auth** is enabled. |
| **TLS Client Key** | The PEM-encoded private key for the client certificate. Required when **Use TLS Client Auth** is enabled. |

## Private data source connect

_Only for Grafana Cloud users._ Private data source connect, or PDC, allows you to establish a private, secured connection between a Grafana Cloud instance, or stack, and data sources secured within a private network. This is useful when your MQTT broker isn't reachable from Grafana Cloud directly. Click the drop-down to locate the URL for PDC.

When PDC is enabled in your Grafana instance, a **Secure SOCKS Proxy** configuration section appears on the data source settings page. For more information, refer to [Private data source connect (PDC)](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/).

## Verify the connection

After you configure the data source, click **Save & test** to verify the connection.

- A successful connection displays the message **MQTT Connected**.
- A failed connection displays the message **MQTT Disconnected**. Check your URI, credentials, and network connectivity, then try again.

## Provision the data source

You can define and configure the MQTT data source using YAML files as part of Grafana's provisioning system. For more information, refer to [Provisioning Grafana](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/administration/provisioning/#data-sources).

```yaml
apiVersion: 1

datasources:
  - name: MQTT
    type: grafana-mqtt-datasource
    jsonData:
      uri: tcp://<BROKER_HOST>:<BROKER_PORT>
      username: <USERNAME>
      clientID: <CLIENT_ID>
      tlsAuth: false
      tlsAuthWithCACert: false
      tlsSkipVerify: false
    secureJsonData:
      password: <PASSWORD>
      tlsCACert: <CA_CERTIFICATE_PEM>
      tlsClientCert: <CLIENT_CERTIFICATE_PEM>
      tlsClientKey: <CLIENT_KEY_PEM>
```

Replace the `<PLACEHOLDER>` values with your broker-specific settings. Omit any `secureJsonData` fields that don't apply to your configuration.

## Provision the data source with Terraform

You can manage the MQTT data source as code using the [Grafana Terraform provider](https://registry.terraform.io/providers/grafana/grafana/latest/docs). For more information, refer to the [Grafana Terraform provider documentation](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/as-code/infrastructure-as-code/terraform/).

```hcl
resource "grafana_data_source" "mqtt" {
  name = "MQTT"
  type = "grafana-mqtt-datasource"

  json_data_encoded = jsonencode({
    uri              = "tcp://<BROKER_HOST>:<BROKER_PORT>"
    username         = "<USERNAME>"
    clientID         = "<CLIENT_ID>"
    tlsAuth          = false
    tlsAuthWithCACert = false
    tlsSkipVerify    = false
  })

  secure_json_data_encoded = jsonencode({
    password      = "<PASSWORD>"
    tlsCACert     = "<CA_CERTIFICATE_PEM>"
    tlsClientCert = "<CLIENT_CERTIFICATE_PEM>"
    tlsClientKey  = "<CLIENT_KEY_PEM>"
  })
}
```

Replace the `<PLACEHOLDER>` values with your broker-specific settings. Omit any `secure_json_data_encoded` fields that don't apply to your configuration.

## Publishing

The data source can publish MQTT messages on behalf of signed-in users, for example from a form panel. Publishing is disabled unless `jsonData.publish.enabled` is `true`.

Permission is granted through Grafana teams: only members of one of the configured teams may publish. Grafana doesn't pass team membership to plugins, so the plugin reads it through the Grafana HTTP API with a service account token. The service account needs permission to read teams (`teams:read`, in Grafana OSS the organization role **Admin**). Membership is matched by login and cached for `teamCacheSeconds`. If the team lookup fails, publishing is refused.

```yaml
jsonData:
  publish:
    enabled: true
    allowedTeams: ['Writers']
    allowedTopics: ['plant/control/+/+'] # MQTT topic filters
    deniedTopics: ['plant/control/raw/#'] # take precedence over allowedTopics
    qos: 1 # default 1
    retain: false # default false
    payloadFormat: envelope # raw (default) or envelope
    origin: grafana # origin in the envelope, default grafana
    maxPayloadBytes: 256 # limit of the published payload, default 256
    teamCacheSeconds: 60 # default 60
    rateLimitPerMinute: 10 # per user, default 10, negative disables
    grafanaUrl: http://localhost:3000 # used for the team lookup, default http://localhost:3000
secureJsonData:
  publishServiceAccountToken: <SERVICE_ACCOUNT_TOKEN>
```

Use an internal `grafanaUrl`. A public URL behind a reverse proxy works, but adds a needless round trip.

### Publish API

```http
POST /api/datasources/uid/<UID>/resources/publish
Content-Type: application/json

{ "topic": "plant/control/boiler/target", "value": "65" }
```

- `topic` must be a concrete topic without wildcards that matches `allowedTopics` and no `deniedTopics`.
- `value` is a string or number without control characters.
- With `payloadFormat: raw`, the value is published as is. With `envelope`, the plugin publishes `{"value":"65","origin":"grafana","user":"<login>"}`. `origin` comes from the configuration and `user` from the signed-in Grafana user, so callers can't set them.
- QoS and retain come from the configuration only.

| Status | Meaning |
| ------ | ------- |
| `200` | `{"published": true}` |
| `400` | Invalid request, topic, or value, or payload too large |
| `403` | Publishing disabled, no signed-in user, user not in an allowed team, or topic not allowed |
| `429` | Rate limit exceeded |
| `503` | Configuration invalid, team lookup failed, or broker unavailable |

Error responses contain `{"published": false, "error": "<reason>"}`. Every attempt is logged with user, topic, value, and result.

`GET /api/datasources/uid/<UID>/resources/publish/permissions` returns `{"canPublish": true}` or `{"canPublish": false, "reason": "..."}`, so panels can disable inputs for users without permission.

**Save & test** also checks the publishing configuration and the team lookup.

## Next steps

- [MQTT query editor](https://grafana.com/docs/plugins/grafana-mqtt-datasource/latest/query-editor/) -- Subscribe to topics and understand how data is formatted.
- [Template variables](https://grafana.com/docs/plugins/grafana-mqtt-datasource/latest/template-variables/) -- Create dynamic dashboards with variable-driven topics.
- [Troubleshoot MQTT data source issues](https://grafana.com/docs/plugins/grafana-mqtt-datasource/latest/troubleshooting/) -- Resolve common connection, authentication, and query problems.
