import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSource } from './datasource';
import { getTemplateSrv } from '@grafana/runtime';
import { MqttDataSourceOptions } from './types';

jest.mock('@grafana/runtime', () => ({
  DataSourceWithBackend: class {},
  getTemplateSrv: jest.fn(),
}));

describe('DataSource', () => {
  const mockReplace = jest.fn().mockImplementation((value) => value);
  (getTemplateSrv as jest.Mock).mockReturnValue({
    replace: mockReplace,
  });

  const scopedVars: ScopedVars = {};
  const mockInstanceSettings = {
    id: 1,
    uid: 'test-uid',
    type: 'mqtt-datasource',
    name: 'Test MQTT',
    meta: {} as any,
    readOnly: false,
    access: 'direct',
    jsonData: {
      uri: 'mqtt://localhost:1883',
      tlsAuth: false,
      tlsAuthWithCACert: false,
      tlsSkipVerify: false,
    },
  } as DataSourceInstanceSettings<MqttDataSourceOptions>;
  let dataSource = new DataSource(mockInstanceSettings);

  const testCases = [
    {
      description: 'should apply base64 URL-safe encoding correctly',
      query: { topic: 'test/topic+/and:more', refId: 'A' },
      expectedResult: 'dGVzdC90b3BpYysvYW5kOm1vcmU', // cspell:disable-line
    },
  ];

  testCases.forEach(({ description, query, expectedResult }) => {
    it(description, () => {
      const result = dataSource.applyTemplateVariables(query, scopedVars);

      expect(mockReplace).toHaveBeenCalledWith(query.topic, scopedVars);
      expect(result.topic).toBe(expectedResult);
    });
  });
});
