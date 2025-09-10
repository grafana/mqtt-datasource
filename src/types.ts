import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface MqttQuery extends DataQuery {
  topic?: string;
  stream?: boolean;
  streamingKey?: string;
}

export interface MqttDataSourceOptions extends DataSourceJsonData {
  uri: string;
  username?: string;
  clientID?: string;
  tlsAuth: boolean;
  tlsAuthWithCACert: boolean;
  tlsSkipVerify: boolean;
}

export interface MqttSecureJsonData {
  password?: string;
  tlsCACert?: string;
  tlsClientKey?: string;
  tlsClientCert?: string;
}
