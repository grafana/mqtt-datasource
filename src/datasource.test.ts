import { ScopedVars } from '@grafana/data';
import { DataSource } from './datasource';
import { getTemplateSrv } from '@grafana/runtime';

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
  let dataSource = new DataSource();

  const testCases = [
    {
      description: 'should apply base64 URL-safe encoding correctly',
      query: { topic: 'test/topic+/and:more' },
      expectedResult: 'dGVzdC90b3BpYysvYW5kOm1vcmU',
    },
  ];

  testCases.forEach(({ description, query, expectedReplaced, expectedResult }) => {
    it(description, () => {
      const result = dataSource.applyTemplateVariables(query, scopedVars);

      expect(mockReplace).toHaveBeenCalledWith(query.topic, scopedVars);
      expect(result.topic).toBe(expectedResult);
    });
  });
});
