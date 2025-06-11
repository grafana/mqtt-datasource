import { test, expect } from '@grafana/plugin-e2e';

test('Smoke test: plugin loads', async ({ createDataSourceConfigPage, page }) => {
  await createDataSourceConfigPage({ type: 'mqtt-datasource' });

  await expect(await page.getByText('Type: MQTT', { exact: true })).toBeVisible();
  await expect(await page.getByPlaceholder('TCP (tcp://), TLS (tls://), or WebSocket (ws://)')).toBeVisible();
});
