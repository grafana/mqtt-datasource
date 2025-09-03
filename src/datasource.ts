import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceInstanceSettings,
  ScopedVars,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MqttDataSourceOptions, MqttQuery } from './types';
import { Observable, from, switchMap } from 'rxjs';
import { getLiveStreamKey } from './streaming';

export class DataSource extends DataSourceWithBackend<MqttQuery, MqttDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MqttDataSourceOptions>) {
    super(instanceSettings);
  }

  query(request: DataQueryRequest<MqttQuery>): Observable<DataQueryResponse> {
    // Add streamingKey to each target using RxJS operators
    return from(
      Promise.all(
        request.targets.map(async (target) => ({
          ...target,
          streamingKey: await getLiveStreamKey(this.uid, target.topic),
        }))
      )
    ).pipe(
      switchMap((updatedTargets) => {
        const updatedRequest = {
          ...request,
          targets: updatedTargets,
        };
        return super.query(updatedRequest);
      })
    );
  }

  applyTemplateVariables(query: MqttQuery, scopedVars: ScopedVars, filters?: any[]): MqttQuery {
    let resolvedTopic = getTemplateSrv().replace(query.topic, scopedVars);
    resolvedTopic = this.base64UrlSafeEncode(resolvedTopic);
    const resolvedQuery: MqttQuery = {
      ...query,
      topic: resolvedTopic,
      refId: query.refId,
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
