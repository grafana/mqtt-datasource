import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MqttDataSourceOptions, MqttQuery } from './types';

export class DataSource extends DataSourceWithBackend<MqttQuery, MqttDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MqttDataSourceOptions>) {
    super(instanceSettings);
  }

  applyTemplateVariables(query: MqttQuery, scopedVars: ScopedVars): Record<string, any> {
    let resolvedTopic = getTemplateSrv().replace(query.topic, scopedVars);
    resolvedTopic = this.base64UrlSafeEncode(resolvedTopic);
    const resolvedQuery: MqttQuery = {
      ...query,
      topic: resolvedTopic,
    };

    return resolvedQuery;
  }

  // There are some restrictions to what characters are allowed to use in a Grafana Live channel:
  //
  //  https://github.com/grafana/grafana-plugin-sdk-go/blob/7470982de35f3b0bb5d17631b4163463153cc204/live/channel.go#L33
  //
  // To comply with these restrictions, the topic is encoded using URL-safe base64 encoding.
  // (RFC 4648; 5. Base 64 Encoding with URL and Filename Safe Alphabet)
  private base64UrlSafeEncode(input: string): string {
    return btoa(input).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }
}
