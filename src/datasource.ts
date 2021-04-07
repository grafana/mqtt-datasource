import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { MqttDataSourceOptions, MqttQuery } from './types';

export class DataSource extends DataSourceWithBackend<MqttQuery, MqttDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MqttDataSourceOptions>) {
    super(instanceSettings);
  }
}
