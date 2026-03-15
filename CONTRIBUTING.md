# Contributing to the MQTT Datasource Plugin

## Prerequisites

- Node.js (v20 or later)
- Go (latest stable version)
- Yarn (v1.22 or later — Yarn classic and Berry are both supported)
- Mage — install after Go is set up:
  ```
  go install github.com/magefile/mage@latest
  ```

## Development Setup

1. Clone the repository
2. Install dependencies:
   ```
   yarn install
   ```
3. Build the frontend:
   ```
   yarn build
   ```
4. Build the Go backend:
   ```
   mage build
   ```

## Development Workflow

The recommended way to develop is with Docker Compose, which starts a pre-configured Grafana instance alongside the plugin automatically:

```
yarn server
```

Then, in separate terminals, start the watchers so code changes are picked up live:

```
yarn dev
```

```
mage watch
```

Start test broker:

```
yarn broker
```

This will start a test MQTT broker on `tcp://localhost:1883`.

Start the test broker with TLS:

```
yarn broker:tls
```

This will start a test MQTT broker on `tls://localhost:8883` with TLS enabled. The TLS certificates are located in the `testdata` folder. If they need to be regenerated, run:

```
yarn broker:pki
```

When testing with the test broker you can subscribe to test data streams using the following topic patterns:

- `millisecond/<number>` - emit data every N milliseconds
- `second/<number>` - emit data every N seconds
- `minute/<number>` - emit data every N minutes
- `hour/<number>` - emit data every N hours

![Test Broker Screenshot](./test_broker.gif)

After making your changes, ensure checks pass:

```
yarn typecheck  # Check TypeScript types
yarn lint       # Lint the Typescript code
yarn test:ci    # Run tests
yarn spellcheck # Run spellcheck
mage test       # Run Go tests
mage lint       # Lint Go code
```

If you've added new functionality, please add appropriate tests.

## Project Structure

- `src/` - Frontend TypeScript/React code
- `pkg/` - Backend Go code
  - `mqtt/` - MQTT client implementation
  - `plugin/` - Grafana plugin implementation
- `scripts/` - Utility scripts
- `testdata/` - Test certificates and data

## Running Against a Local Grafana Instance

> **Most contributors should use `yarn server` instead.** Docker Compose is the recommended development setup — it starts a pre-configured Grafana instance with no manual setup required.

This section is for the specific case where you already have a Grafana installation running on your machine and want to load the locally-built plugin into it (e.g., you are testing the plugin against a specific Grafana version or configuration that you manage yourself). If that does not describe your situation, use `yarn server` instead.

### 1. Build the plugin

```
yarn build
mage build
```

### 2. Copy the built plugin to Grafana's plugins directory

```
cp -r . /var/lib/grafana/plugins/mqtt-datasource
```

Adjust the destination path if your Grafana installation uses a different plugins directory (check the `plugins` setting in your `grafana.ini`).

### 3. Allow the unsigned plugin in Grafana

Local builds are not signed, so you must explicitly permit the plugin in `grafana.ini`:

```ini
[paths]
plugins = /var/lib/grafana/plugins

[plugins]
allow_loading_unsigned_plugins = grafana-mqtt-datasource
```

On Linux the config file is typically at `/etc/grafana/grafana.ini`. On macOS (Homebrew) it is usually at `/usr/local/etc/grafana/grafana.ini`.

### 4. Restart Grafana and verify

Restart the Grafana service, then open the Grafana UI and navigate to **Administration → Plugins** to confirm the MQTT datasource appears and is enabled.

## Submitting PR

If you are creating a PR, ensure to run `yarn changeset` from your branch. Provide the details accordingly. It will create `*.md` file inside `./.changeset` folder. Later during the release, based on these changesets, package version will be bumped and changelog will be generated.

## Releasing & Bumping version

To create a new release, execute `yarn changeset version`. This will update the Changelog and bump the version in `package.json` file. Commit those changes.
