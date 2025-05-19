import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MqttDataSourceOptions, MqttQuery } from './types';

export class DataSource extends DataSourceWithBackend<MqttQuery, MqttDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MqttDataSourceOptions>) {
    super(instanceSettings);
  }

  applyTemplateVariables(query: MqttQuery, scopedVars: ScopedVars, filters?: any[]): MqttQuery {
    let resolvedTopic = getTemplateSrv().replace(query.topic, scopedVars);
    resolvedTopic = resolvedTopic.replace(/\+/gi, '__PLUS__');
    resolvedTopic = resolvedTopic.replace(/\#/gi, '__HASH__');
    const resolvedQuery: MqttQuery = {
      ...query,
      topic: resolvedTopic,
      refId: query.refId,
    };

    return resolvedQuery;
  }
}
