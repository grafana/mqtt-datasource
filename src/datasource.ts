import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MqttDataSourceOptions, MqttQuery } from './types';

export class DataSource extends DataSourceWithBackend<MqttQuery, MqttDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MqttDataSourceOptions>) {
    super(instanceSettings);
  }

  applyTemplateVariables(query: MqttQuery, scopedVars: ScopedVars): Record<string, any> {
    const resolvedQuery: MqttQuery = {
      ...query,
      topic: getTemplateSrv().replace(query.topic, scopedVars),
    };

    return resolvedQuery;
  }
}
