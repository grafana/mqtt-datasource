import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MqttQuery extends DataQuery {
  topic?: string;
  stream?: boolean;
}

export interface MqttDataSourceOptions extends DataSourceJsonData {
  uri: string;
  username?: string;
}

export interface MqttSecureJsonData {
  password?: string;
}
