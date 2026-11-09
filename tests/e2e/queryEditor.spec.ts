/// <reference types="node" />
import { expect, test } from '@grafana/plugin-e2e';
import { type Locator, type Page } from '@playwright/test';

const PLUGIN_TYPE = 'grafana-mqtt-datasource';
const PROVISIONING_FILE = 'datasources.yml';

// GRAFANA_URL is set only by the Cloud cron workflow. Its presence indicates the
// local provisioning file is not applied.
const isCloudRun = !!process.env.GRAFANA_URL;

const CLOUD_DEFAULT_UID = 'mqtt-ds-m';
const DATASOURCE_UID = process.env.DS_E2E_UID || (isCloudRun ? CLOUD_DEFAULT_UID : 'mqtt');

// Topic published by the local mqtt-publisher service and by the Cloud data generator.
const MQTT_TOPIC = process.env.DS_INSTANCE_TOPIC ?? 'grafana/world_data';

// MQTT is a live stream with no stored retention, so a relative window is correct
// (unlike seeded historical fixtures, which need a pinned ISO range).
const RANGE_FROM = 'now-15m';
const RANGE_TO = 'now';

function exploreUrl(uid: string, opts?: { topic?: string }): string {
  const query: Record<string, unknown> = {
    refId: 'A',
    datasource: { type: PLUGIN_TYPE, uid },
  };
  if (opts?.topic) {
    query.topic = opts.topic;
  }
  const panes = JSON.stringify({
    explore: {
      datasource: uid,
      queries: [query],
      range: { from: RANGE_FROM, to: RANGE_TO },
    },
  });
  return `/explore?orgId=1&schemaVersion=1&panes=${encodeURIComponent(panes)}`;
}

// Grafana 13 migrated query editor row selectors from aria-label to data-testid
// (https://github.com/grafana/grafana/pull/121784). Match both shapes.
function getQueryEditorRow(page: Page, refId: string): Locator {
  return page
    .locator('[data-testid="data-testid Query editor row"], [aria-label="Query editor row"]')
    .filter({
      has: page.locator(
        `[data-testid="data-testid Query editor row title ${refId}"], [aria-label="Query editor row title ${refId}"]`
      ),
    })
    .or(page.getByRole('button', { name: `Query editor row title ${refId}` }).locator('..'));
}

function topicInput(page: Page): Locator {
  return page.getByPlaceholder('e.g. "home/bedroom/temperature"');
}

// TODO: remove once @grafana/plugin-e2e exposes body reading natively.
// Not async: callers destructure { responsePromise, getBody } synchronously.
function waitForQueryDataResponseWithBody(page: Page) {
  let body: Record<string, unknown> | null = null;
  const responsePromise = page.waitForResponse(async (r) => {
    if (!r.url().includes('/api/ds/query') || !r.ok()) {
      return false;
    }
    const b: any = await r.json().catch(() => null);
    if (!Array.isArray(b?.results?.A?.frames)) {
      return false;
    }
    body = b;
    return true;
  });
  return { responsePromise, getBody: () => body };
}

test.describe('Query editor', () => {
  test.describe('rendering', () => {
    test(
      'smoke: renders Topic field',
      { tag: '@plugins' },
      async ({ page, readProvisionedDataSource }) => {
        const uid = isCloudRun
          ? DATASOURCE_UID
          : (await readProvisionedDataSource({ fileName: PROVISIONING_FILE })).uid;
        await page.goto(exploreUrl(uid));

        const queryRow = getQueryEditorRow(page, 'A');
        await expect(queryRow.getByText('Topic', { exact: true })).toBeVisible();
        await expect(topicInput(page)).toBeVisible();
      }
    );

    test('can enter a topic', async ({ page, readProvisionedDataSource }) => {
      const uid = isCloudRun
        ? DATASOURCE_UID
        : (await readProvisionedDataSource({ fileName: PROVISIONING_FILE })).uid;
      await page.goto(exploreUrl(uid));

      await topicInput(page).fill('home/bedroom/temperature');
      await expect(topicInput(page)).toHaveValue('home/bedroom/temperature');
    });
  });

  test.describe('query execution', () => {
    test('receives a streaming channel response', async ({
      page,
      explorePage,
      readProvisionedDataSource,
    }) => {
      const uid = isCloudRun
        ? DATASOURCE_UID
        : (await readProvisionedDataSource({ fileName: PROVISIONING_FILE })).uid;

      await explorePage.mockQueryDataResponse({
        results: {
          A: {
            frames: [
              {
                schema: {
                  meta: { channel: `ds/${uid}/1s/mock-topic` },
                  fields: [],
                },
                data: { values: [] },
              },
            ],
          },
        },
      });

      const { responsePromise, getBody } = waitForQueryDataResponseWithBody(page);
      await page.goto(exploreUrl(uid, { topic: MQTT_TOPIC }));
      await responsePromise;

      const body = getBody() as any;
      expect(body?.results?.A?.frames?.length).toBeGreaterThan(0);
      expect(body.results.A.frames[0].schema?.meta?.channel).toContain(`ds/${uid}/`);
    });
  });
});

test.describe('Query editor with fixture data', () => {
  test.describe.configure({ mode: 'serial' });

  test.describe('grafana/world_data', () => {
    test('query returns a live channel', async ({ page, readProvisionedDataSource }) => {
      const uid = isCloudRun
        ? DATASOURCE_UID
        : (await readProvisionedDataSource({ fileName: PROVISIONING_FILE })).uid;

      const { responsePromise, getBody } = waitForQueryDataResponseWithBody(page);
      await page.goto(exploreUrl(uid, { topic: MQTT_TOPIC }));
      await responsePromise;

      const body = getBody() as any;
      expect(body?.results?.A?.error).toBeUndefined();
      expect(body?.results?.A?.frames?.length).toBeGreaterThan(0);

      const channel = body.results.A.frames[0].schema?.meta?.channel as string | undefined;
      expect(channel).toBeTruthy();
      expect(channel).toMatch(/^ds\//);
    });

    test('topic field is populated from the Explore URL', async ({
      page,
      readProvisionedDataSource,
    }) => {
      const uid = isCloudRun
        ? DATASOURCE_UID
        : (await readProvisionedDataSource({ fileName: PROVISIONING_FILE })).uid;

      await page.goto(exploreUrl(uid, { topic: MQTT_TOPIC }));
      await expect(topicInput(page)).toHaveValue(MQTT_TOPIC);
    });
  });
});
