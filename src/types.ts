import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MqttQuery extends DataQuery {
  queryText?: string;
  stream?: boolean;
}

export interface MqttDataSourceOptions extends DataSourceJsonData {
  host: string;
  port: number;
  username?: string;
}

export interface MqttSecureJsonData {
  password?: string;
}
