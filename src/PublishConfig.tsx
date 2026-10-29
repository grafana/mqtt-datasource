import React from 'react';

import { DataSourcePluginOptionsEditorProps, onUpdateDatasourceSecureJsonDataOption } from '@grafana/data';
import { ConfigSection } from '@grafana/plugin-ui';
import { Field, Input, RadioButtonGroup, SecretInput, Switch, TagsInput } from '@grafana/ui';
import { MqttDataSourceOptions, MqttPublishOptions, MqttSecureJsonData } from './types';

type Props = DataSourcePluginOptionsEditorProps<MqttDataSourceOptions, MqttSecureJsonData>;

const WIDTH_LONG = 40;
const WIDTH_SHORT = 16;

const payloadFormatOptions = [
  { label: 'Raw value', value: 'raw' as const },
  { label: 'JSON envelope', value: 'envelope' as const },
];

const qosOptions = [0, 1, 2].map((qos) => ({ label: String(qos), value: qos }));

export const PublishConfig = (props: Props) => {
  const { options, onOptionsChange } = props;
  const publish: MqttPublishOptions = options.jsonData.publish ?? {};

  const update = <K extends keyof MqttPublishOptions>(key: K, value: MqttPublishOptions[K]) => {
    onOptionsChange({
      ...options,
      jsonData: { ...options.jsonData, publish: { ...publish, [key]: value } },
    });
  };

  const updateNumber = (key: 'maxPayloadBytes' | 'teamCacheSeconds' | 'rateLimitPerMinute') => {
    return (event: React.FormEvent<HTMLInputElement>) => {
      const text = event.currentTarget.value;
      update(key, text === '' ? undefined : Number(text));
    };
  };

  const onResetToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: { ...options.secureJsonFields, publishServiceAccountToken: false },
      secureJsonData: { ...options.secureJsonData, publishServiceAccountToken: '' },
    });
  };

  return (
    <ConfigSection
      title="Publishing"
      description="Allow members of selected Grafana teams to publish MQTT messages through the datasource resource API."
      isCollapsible
      isInitiallyOpen={publish.enabled === true}
    >
      <Field label="Enable publishing">
        <Switch value={publish.enabled ?? false} onChange={(e) => update('enabled', e.currentTarget.checked)} />
      </Field>

      {publish.enabled ? (
        <>
          <Field label="Allowed teams" description="Grafana teams whose members may publish." required>
            <TagsInput
              tags={publish.allowedTeams ?? []}
              onChange={(tags) => update('allowedTeams', tags)}
              width={WIDTH_LONG}
            />
          </Field>

          <Field
            label="Service account token"
            description="Token of a Grafana service account allowed to read teams (teams:read)."
            required
          >
            <SecretInput
              width={WIDTH_LONG}
              isConfigured={options.secureJsonFields?.publishServiceAccountToken ?? false}
              onReset={onResetToken}
              onBlur={onUpdateDatasourceSecureJsonDataOption(props, 'publishServiceAccountToken')}
            />
          </Field>

          <Field label="Grafana URL" description="Used to read team membership. Default: http://localhost:3000">
            <Input
              width={WIDTH_LONG}
              value={publish.grafanaUrl ?? ''}
              placeholder="http://localhost:3000"
              onChange={(e) => update('grafanaUrl', e.currentTarget.value)}
            />
          </Field>

          <Field label="Allowed topics" description="MQTT topic filters (+ and # wildcards)." required>
            <TagsInput
              tags={publish.allowedTopics ?? []}
              onChange={(tags) => update('allowedTopics', tags)}
              width={WIDTH_LONG}
            />
          </Field>

          <Field label="Denied topics" description="Topic filters that are never allowed; they take precedence.">
            <TagsInput
              tags={publish.deniedTopics ?? []}
              onChange={(tags) => update('deniedTopics', tags)}
              width={WIDTH_LONG}
            />
          </Field>

          <Field
            label="Payload format"
            description="Raw publishes the value as is. The JSON envelope adds origin and the signed-in user."
          >
            <RadioButtonGroup
              options={payloadFormatOptions}
              value={publish.payloadFormat ?? 'raw'}
              onChange={(value) => update('payloadFormat', value)}
            />
          </Field>

          {publish.payloadFormat === 'envelope' ? (
            <Field label="Origin" description="Value of the origin field in the envelope. Default: grafana">
              <Input
                width={WIDTH_SHORT}
                value={publish.origin ?? ''}
                placeholder="grafana"
                onChange={(e) => update('origin', e.currentTarget.value)}
              />
            </Field>
          ) : null}

          <Field label="QoS">
            <RadioButtonGroup
              options={qosOptions}
              value={publish.qos ?? 1}
              onChange={(value) => update('qos', value)}
            />
          </Field>

          <Field label="Retain">
            <Switch value={publish.retain ?? false} onChange={(e) => update('retain', e.currentTarget.checked)} />
          </Field>

          <Field label="Max payload bytes" description="Default: 256">
            <Input
              type="number"
              width={WIDTH_SHORT}
              value={publish.maxPayloadBytes ?? ''}
              placeholder="256"
              onChange={updateNumber('maxPayloadBytes')}
            />
          </Field>

          <Field label="Team cache seconds" description="How long team membership is cached. Default: 60">
            <Input
              type="number"
              width={WIDTH_SHORT}
              value={publish.teamCacheSeconds ?? ''}
              placeholder="60"
              onChange={updateNumber('teamCacheSeconds')}
            />
          </Field>

          <Field label="Rate limit per minute" description="Publish requests per user. Default: 10, negative disables.">
            <Input
              type="number"
              width={WIDTH_SHORT}
              value={publish.rateLimitPerMinute ?? ''}
              placeholder="10"
              onChange={updateNumber('rateLimitPerMinute')}
            />
          </Field>
        </>
      ) : null}
    </ConfigSection>
  );
};
