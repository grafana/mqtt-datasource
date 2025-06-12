import { expect, test } from '@grafana/plugin-e2e';

test('Smoke test: plugin loads', async ({ createDataSourceConfigPage, page }) => {
  await createDataSourceConfigPage({ type: 'grafana-mqtt-datasource' });

  await expect(await page.getByText('Type: MQTT', { exact: true })).toBeVisible();
  await expect(await page.getByText('URI *', { exact: true })).toBeVisible();
});
