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
last_reviewed: 2026-02-20
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
| **URI** | The URI of your MQTT broker. Include the scheme and port. Supported schemes: `tcp://` (unencrypted, default port `1883`), `tls://` (TLS-encrypted, default port `8883`), `ws://` (WebSocket, default port `80`), and `wss://` (WebSocket Secure, default port `443`). For example, `tcp://localhost:1883` or `tls://broker.example.com:8883`. If you omit the port, the default for the scheme is used. |
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
| **CA Cert** | The PEM-encoded CA certificate used to verify the broker's server certificate. Required when **With CA Cert** is enabled. |
| **Client Cert** | The PEM-encoded client certificate for mTLS authentication. Required when **Use TLS Client Auth** is enabled. |
| **Client Key** | The PEM-encoded private key for the client certificate. Required when **Use TLS Client Auth** is enabled. |

## Private data source connect

If your MQTT broker is on a private network that isn't reachable from Grafana Cloud, you can use [Private data source connect (PDC)](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/) to route the connection through a secure SOCKS proxy.

When PDC is enabled in your Grafana instance, a **Secure SOCKS Proxy** configuration section appears on the data source settings page.

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
