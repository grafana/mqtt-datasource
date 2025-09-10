# Contributing to the MQTT Datasource Plugin

## Prerequisites

- Node.js (v20 or later)
- Go (latest stable version)
- Yarn (v1.22.x)
- Mage

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

Start watching the frontend and backend code for changes:

```
yarn dev
```

and in another terminal:

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

## Submitting PR

If you are creating a PR, ensure to run `yarn changeset` from your branch. Provide the details accordingly. It will create `*.md` file inside `./.changeset` folder. Later during the release, based on these changesets, package version will be bumped and changelog will be generated.

## Releasing & Bumping version

To create a new release, execute `yarn changeset version`. This will update the Changelog and bump the version in `package.json` file. Commit those changes.
