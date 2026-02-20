---
aliases:
  - /docs/plugins/grafana-mqtt-datasource/troubleshooting/
description: Troubleshoot common issues with the MQTT data source in Grafana.
keywords:
  - grafana
  - mqtt
  - troubleshooting
  - errors
  - connection
  - authentication
labels:
  products:
    - cloud
    - enterprise
    - oss
menuTitle: Troubleshooting
title: Troubleshoot MQTT data source issues
weight: 500
last_reviewed: 2026-02-20
---

# Troubleshoot MQTT data source issues

This document provides solutions to common issues you may encounter when configuring or using the MQTT data source. For configuration instructions, refer to [Configure the MQTT data source](https://grafana.com/docs/plugins/grafana-mqtt-datasource/latest/configure/).

## Connection errors

These errors occur when Grafana can't connect to the MQTT broker.

### "MQTT Disconnected" on Save & test

**Symptoms:**

- Clicking **Save & test** displays **MQTT Disconnected**.
- Panels show no data.

**Possible causes and solutions:**

| Cause | Solution |
|-------|----------|
| Incorrect URI | Verify the URI includes the correct scheme (`tcp://`, `tls://`, or `ws://`), host, and port. For example, `tcp://localhost:1883`. |
| Broker not running | Confirm the MQTT broker is running and accepting connections on the specified port. |
| Firewall or network restrictions | Ensure the Grafana server can reach the broker's host and port. Check firewall rules and security groups. |
| Wrong scheme for TLS | If the broker requires TLS, use `tls://` instead of `tcp://`. |
| DNS resolution failure | Verify the broker hostname resolves correctly from the Grafana server. |

### Connection drops or intermittent disconnections

**Symptoms:**

- Panels stop updating and then resume after a delay.
- Grafana logs show "MQTT Connection lost" warnings.

**Solutions:**

1. Check the MQTT broker logs for client evictions or connection limits.
1. Ensure only one Grafana instance uses each **Client ID**. If multiple instances share a Client ID, the broker disconnects the older connection.
1. Verify network stability between the Grafana server and the broker.
1. The plugin automatically reconnects after a disconnection with a maximum interval of 10 seconds.

## Authentication errors

These errors occur when credentials or TLS certificates are invalid.

### "error connecting to MQTT broker" with credentials

**Symptoms:**

- **Save & test** fails with an error mentioning connection failure.
- Broker logs show authentication rejections.

**Possible causes and solutions:**

| Cause | Solution |
|-------|----------|
| Wrong username or password | Re-enter the credentials in the data source configuration. Passwords are stored securely and can't be viewed after saving -- click the reset button and enter the password again. |
| Missing credentials | Some brokers reject anonymous connections. Add a **Username** and **Password** if required. |
| Account disabled or expired | Verify the account is active in your MQTT broker's user management system. |

### TLS certificate errors

**Symptoms:**

- Connection fails when using `tls://` scheme.
- Error messages mention certificate verification, handshake failure, or key mismatch.

**Possible causes and solutions:**

| Cause | Solution |
|-------|----------|
| Self-signed server certificate | Enable **With CA Cert** and paste your CA certificate, or enable **Skip TLS Verification** for testing. |
| Expired certificate | Check expiration dates on your CA, client, and server certificates. Renew any expired certificates. |
| Mismatched client cert and key | Ensure the **Client Cert** and **Client Key** are a matching pair. Regenerate them if necessary. |
| Wrong CA certificate | Verify the **CA Cert** is the certificate that signed the broker's server certificate. |
| Certificate format | Ensure certificates are PEM-encoded (starting with `-----BEGIN CERTIFICATE-----`). |

{{< admonition type="caution" >}}
Enabling **Skip TLS Verification** disables certificate chain and hostname verification. Use this only for testing, not in production environments.
{{< /admonition >}}

## Query errors

These errors occur when subscribing to topics or processing messages.

### No data in panels

**Symptoms:**

- The panel shows "No data" despite the data source being connected.
- The connection test shows **MQTT Connected** but panels are empty.

**Possible causes and solutions:**

| Cause | Solution |
|-------|----------|
| Empty topic | Ensure the **Topic** field in the query editor isn't blank. |
| No messages on topic | Verify that messages are being published to the topic using an MQTT client tool such as `mosquitto_sub`. |
| Wrong topic path | Double-check the topic path for typos. MQTT topics are case-sensitive. |
| Panel time range | The time range selector doesn't affect MQTT streaming. Data only appears while the panel is open and receiving messages. |
| Retained messages only | If the topic only has a retained message, it appears once on subscription. New data requires new messages to be published. |

### Unexpected field types or missing fields

**Symptoms:**

- Numeric values appear as strings.
- JSON object keys are missing from the data frame.

**Solutions:**

1. Ensure numeric values are published without quotes (for example, `23.5` not `"23.5"`).
1. For JSON payloads, only top-level keys are extracted into separate fields. Nested objects appear as JSON-typed fields. Use the **Extract fields** transformation to access nested values.
1. If the first message defines a field as one type and a later message sends a different type for the same key, the later value is dropped. Ensure consistent data types across messages.

## Performance issues

These issues relate to high message volumes or resource usage.

### High CPU or memory usage

**Symptoms:**

- Grafana uses excessive resources when many MQTT topics are active.
- Dashboard panels lag or become unresponsive.

**Solutions:**

1. Reduce the number of subscribed topics per dashboard.
1. Increase the query interval to reduce the frequency of data frame generation. A longer interval buffers more messages per push but reduces processing overhead.
1. Avoid subscribing to broad wildcard topics (such as `#`) that match large numbers of subtopics.
1. Close dashboards and panels you aren't actively using, since each open panel maintains an active subscription.

## Enable debug logging

To capture detailed error information for troubleshooting:

1. Set the Grafana log level to `debug` in the configuration file:

   ```ini
   [log]
   level = debug
   ```

1. Restart Grafana for the change to take effect.
1. Review logs in `/var/log/grafana/grafana.log` (or your configured log location).
1. Look for entries with "MQTT" that include connection, subscription, and message details.
1. Reset the log level to `info` after troubleshooting to avoid excessive log volume.

## Get additional help

If you've tried the solutions in this document and still encounter issues:

1. Check the [Grafana community forums](https://community.grafana.com/) for similar issues and discussions.
1. Review the [MQTT datasource GitHub issues](https://github.com/grafana/mqtt-datasource/issues) for known bugs and feature requests.
1. Consult the [MQTT v3.1.1 specification](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) for protocol-level guidance.
1. Contact [Grafana Support](https://grafana.com/support/) if you're a Cloud Pro, Cloud Advanced, or Enterprise customer.
1. When reporting issues, include:
   - Grafana version and plugin version
   - Error messages (redact credentials and sensitive information)
   - Steps to reproduce the issue
   - Relevant broker configuration (redact credentials)
