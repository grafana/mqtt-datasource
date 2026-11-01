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
  publish?: MqttPublishOptions;
}

export interface MqttPublishOptions {
  enabled?: boolean;
  allowedTeams?: string[];
  allowedTopics?: string[];
  deniedTopics?: string[];
  qos?: number;
  retain?: boolean;
  payloadFormat?: 'raw' | 'envelope';
  origin?: string;
  maxPayloadBytes?: number;
  teamCacheSeconds?: number;
  grafanaUrl?: string;
  rateLimitPerMinute?: number;
}

export interface MqttSecureJsonData {
  password?: string;
  tlsCACert?: string;
  tlsClientKey?: string;
  tlsClientCert?: string;
  publishServiceAccountToken?: string;
}
