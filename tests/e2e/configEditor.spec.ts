import { expect, test } from '@grafana/plugin-e2e';
import { type Page } from '@playwright/test';

import { MqttDataSourceOptions } from '../../src/types';

const PLUGIN_TYPE = 'grafana-mqtt-datasource';
const PROVISIONING_FILE = 'datasources.yml';

// GRAFANA_URL is set only by the Cloud cron workflow (playwright-cloud). Local and PR CI
// don't set it, so its presence is a reliable signal that we're running against a shared
// Cloud instance where the local provisioning/datasources/datasources.yml is not applied.
const isCloudRun = !!process.env.GRAFANA_URL;

// Cloud-managed datasource uid follows `{resourceName}-ds-m` (infra/grafana/utils.ts).
const CLOUD_DEFAULT_UID = 'mqtt-ds-m';

const mqttHost = process.env.DS_INSTANCE_HOST ?? 'mqtt-broker';
const mqttPort = process.env.DS_INSTANCE_PORT ?? '1883';
const mqttUri = process.env.DS_INSTANCE_URI ?? `tcp://${mqttHost}:${mqttPort}`;

async function configurePDC(page: Page, networkName: string) {
  await page.getByRole('combobox', { name: 'Private data source connect' }).click();
  await page.getByText(networkName).click();
}

test.describe('Config editor', () => {
  test.describe('rendering', () => {
    test(
      'smoke: should render config editor',
      { tag: '@plugins' },
      async ({ createDataSourceConfigPage, page }) => {
        await createDataSourceConfigPage({ type: PLUGIN_TYPE });

        await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
        await expect(page.getByText('URI *', { exact: true })).toBeVisible();
      }
    );

    test('should render Connection section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: PLUGIN_TYPE });

      await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
      await expect(
        page.getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')
      ).toBeVisible();
      await expect(page.getByText('Client ID', { exact: true })).toBeVisible();
      await expect(page.getByText('If not set, a random client ID is used.')).toBeVisible();
    });

    test('should render Authentication section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: PLUGIN_TYPE });

      const heading = page.getByRole('heading', { name: 'Authentication', exact: true });
      await heading.scrollIntoViewIfNeeded();
      await expect(heading).toBeVisible();
      await expect(page.getByPlaceholder('Username')).toBeVisible();
      await expect(page.getByPlaceholder('Password')).toBeVisible();
      await expect(page.getByText('Use TLS Client Auth', { exact: true })).toBeVisible();
      await expect(page.getByText('Skip TLS Verification', { exact: true })).toBeVisible();
      await expect(page.getByText('With CA Cert', { exact: true })).toBeVisible();
    });

    test('should render TLS Configuration when CA cert is enabled', async ({
      createDataSourceConfigPage,
      page,
    }) => {
      await createDataSourceConfigPage({ type: PLUGIN_TYPE });

      // The TLS switches have no accessible name; the wrapping label intercepts
      // pointer events, so force the click. Index 3 is "With CA Cert"
      // (Default, Use TLS Client Auth, Skip TLS Verification, With CA Cert).
      await page.getByRole('switch').nth(3).click({ force: true });
      await expect(page.getByRole('heading', { name: 'TLS Configuration', exact: true })).toBeVisible();
      await expect(page.getByText('TLS CA Certificate')).toBeVisible();
    });
  });

  test.describe('provisioned datasource', () => {
    test.beforeEach(() => {
      test.skip(
        isCloudRun,
        'Provisioned-datasource tests assert values from the local provisioning YAML, which is not applied on the shared Cloud instance.'
      );
    });

    test('should load provisioned URI', async ({
      readProvisionedDataSource,
      gotoDataSourceConfigPage,
      page,
    }) => {
      const ds = await readProvisionedDataSource<MqttDataSourceOptions>({
        fileName: PROVISIONING_FILE,
      });
      await gotoDataSourceConfigPage(ds.uid);

      await expect(
        page.getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')
      ).toHaveValue(ds.jsonData.uri);
    });

    test('should load provisioned TLS switches as off', async ({
      readProvisionedDataSource,
      gotoDataSourceConfigPage,
      page,
    }) => {
      const ds = await readProvisionedDataSource<MqttDataSourceOptions>({
        fileName: PROVISIONING_FILE,
      });
      await gotoDataSourceConfigPage(ds.uid);

      // TLS Configuration is only rendered when a TLS switch is on.
      await expect(page.getByRole('heading', { name: 'TLS Configuration', exact: true })).toBeHidden();
      await expect(page.getByText('TLS CA Certificate')).toHaveCount(0);
    });
  });

  test.describe('save & test', () => {
    test('should pass health check for provisioned datasource', async ({
      readProvisionedDataSource,
      gotoDataSourceConfigPage,
      page,
    }) => {
      const uid = isCloudRun
        ? process.env.DS_E2E_UID || CLOUD_DEFAULT_UID
        : (await readProvisionedDataSource<MqttDataSourceOptions>({ fileName: PROVISIONING_FILE }))
            .uid;
      const configPage = await gotoDataSourceConfigPage(uid);

      // Match both `Save & test` (editable: true) and `Test` (editable: false)
      await page.getByRole('button', { name: /^(Save & test|Test)$/ }).click();
      await expect(configPage).toHaveAlert('success');
      await expect(page.getByText('MQTT Connected')).toBeVisible();
    });

    test('should show error alert when health check fails', async ({
      createDataSourceConfigPage,
      page,
    }) => {
      const configPage = await createDataSourceConfigPage({ type: PLUGIN_TYPE });

      await page
        .getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')
        .fill(mqttUri);
      await configPage.mockHealthCheckResponse({ status: 'ERROR', message: 'MQTT Disconnected' }, 400);

      await configPage.saveAndTest();
      await expect(configPage).toHaveAlert('error');
    });

    test('should show error alert when backend is unreachable', async ({
      createDataSourceConfigPage,
      page,
    }) => {
      const configPage = await createDataSourceConfigPage({ type: PLUGIN_TYPE });

      // `localhost` from inside the Grafana container never resolves to the MQTT broker
      await page
        .getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')
        .fill('tcp://localhost:1883');
      await page.getByRole('button', { name: /^(Save & test|Test)$/ }).click();
      await expect(configPage).toHaveAlert('error');
    });

    test('valid credentials should display a success alert on the page', async ({
      createDataSourceConfigPage,
      page,
    }) => {
      test.skip(
        !process.env.CI && !process.env.DS_INSTANCE_HOST,
        'MQTT broker must be reachable from inside Grafana; set DS_INSTANCE_HOST or run in CI'
      );
      test.skip(
        isCloudRun,
        'Ad-hoc save & test connectivity is not reliable on the shared Cloud instance; covered by the provisioned health check.'
      );

      const configPage = await createDataSourceConfigPage({ type: PLUGIN_TYPE });
      await page
        .getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')
        .fill(mqttUri);
      await page.getByPlaceholder('Username').fill(process.env.DS_INSTANCE_USERNAME ?? '');
      await page.getByPlaceholder('Password').fill(process.env.DS_INSTANCE_PASSWORD ?? '');

      if (process.env.DS_PDC_NETWORK_NAME) {
        await configurePDC(page, process.env.DS_PDC_NETWORK_NAME);
      }

      await configPage.saveAndTest();
      await expect(configPage).toHaveAlert('success', { hasNotText: 'Datasource updated' });
    });
  });
});
