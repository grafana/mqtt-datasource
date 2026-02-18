# MQTT data source for Grafana

The MQTT data source plugin allows you to visualize streaming MQTT data from within Grafana.

## Disclaimer

This plugin does not provide a storage for your events. It means, that if you want to query your historical data - this plugin is not for you. This datasource provides you an access to MQTT connection, which could be used to get Retained topics, but NOT intended to work with historical data. If you want to work with historical data please take a look for some storage engine like [Loki](https://github.com/grafana/loki) to store events that you receive from MQTT.

## Requirements

The MQTT data source has the following requirements:

- Grafana user with a server or organization administration role; refer to [Permissions](https://grafana.com/docs/grafana/latest/permissions/).
- Access to a MQTT broker.

## Configure the data source

[Add a data source](https://grafana.com/docs/grafana/latest/datasources/add-a-data-source/) by filling in the following fields:

#### Basic fields

| Field       | Description                                                                                                         |
| ----------- | ------------------------------------------------------------------------------------------------------------------- |
| Name        | A name for this particular MQTT data source                                                                         |
| URI         | The scheme, host, and port of the MQTT Broker. Supported schemes: TCP (tcp://), TLS (tls://), and WebSocket (ws://) |
| Client ID   | (Optional) The client ID to use when connecting to the MQTT broker                                                  |

#### Authentication fields

| Field    | Description                                                       |
| -------- | ----------------------------------------------------------------- |
| Username | (Optional) The username to use when connecting to the MQTT broker |
| Password | (Optional) The password to use when connecting to the MQTT broker |

### Private data source connect (PDC)

{{< admonition type="note" >}}
Private data source connect (PDC) is only available for Grafana Cloud users.
{{< /admonition >}}

Use PDC to connect to and query data within a secure network without opening that network to inbound traffic from Grafana Cloud. For more information on how PDC works, refer to [Private data source connect](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/). For setup instructions, refer to [Configure Grafana private data source connect (PDC)](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/configure-pdc/).

- **Private data source connect** - Select the PDC connection from the drop-down menu or create a new connection.

## Query the data source

The query editor allows you to specify which MQTT topics the panel will subscribe to. Refer to the [MQTT v3.1.1 specification](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc398718106)
for more information about valid topic names and filters.

![mqtt dashboard](./test_broker.gif)

## Known limitations

- The plugin currently does not support all of the MQTT CONNECT packet options.
- This plugin automatically supports topics publishing numbers, strings, booleans, and JSON formatted values. Nested object values can be extracted using the `Extract Fields` transformation.
- This plugin automatically attaches timestamps to the messages when they are received. Timestamps included in the message body can be parsed using the `Convert field type` transformation.

## Install the plugin

### Installation Pre-requisites

Refer to: [Building a Streaming Datasource Backend Plugin](https://grafana.com/tutorials/build-a-streaming-data-source-plugin/)

Details: [Ubuntu](https://github.com/grafana/mqtt-datasource/issues/15#issuecomment-894477802) [Windows](https://github.com/grafana/mqtt-datasource/issues/15#issuecomment-894534196)

### Meet compatibility requirements

This plugin currently supports MQTT v3.1.x.

**Note: Since this plugin uses the Grafana Live Streaming API, make sure to use Grafana v8.0+**

### Installation Steps

1. Clone the plugin to your Grafana plugins directory.
2. Build the plugin by running `yarn install` and then `yarn build`.

NOTE: The `yarn build` command above might fail on a non-unix-like system, like Windows, where you can try replacing the `rm -rf` command with `rimraf` in the `./package.json` file to make it work.

3. Run `mage reloadPlugin` or restart Grafana for the plugin to load.

### Verify that the plugin is installed

1. In Grafana from the left-hand menu, navigate to **Configuration** > **Data sources**.
2. From the top-right corner, click the **Add data source** button.
3. Search for `MQTT` in the search field, and hover over the MQTT search result.
4. Click the **Select** button for MQTT.
