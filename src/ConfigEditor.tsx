import React, { ChangeEvent } from 'react';
import { InlineField, InlineFieldRow, FieldSet, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MqttDataSourceOptions, MqttSecureJsonData } from './types';
import { handlerFactory } from './handleEvent';

interface Props extends DataSourcePluginOptionsEditorProps<MqttDataSourceOptions, MqttSecureJsonData> {}

export const ConfigEditor = (props: Props) => {
  const {
    onOptionsChange,
    options,
    options: { jsonData, secureJsonData, secureJsonFields },
  } = props;
  const { uri, username } = jsonData;
  const handleChange = handlerFactory(options, onOptionsChange);

  const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        password: event.target.value,
      },
    });
  };

  const onResetPassword = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        password: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        password: '',
      },
    });
  };

  return (
    <>
      <FieldSet label="Connection">
        <InlineFieldRow>
          <InlineField
            label="URI"
            labelWidth={10}
            tooltip="Supported schemes: TCP (tcp://), TLS (tls://), or WebSocket (ws://)"
          >
            <Input
              width={30}
              name="uri"
              required
              value={uri}
              autoComplete="off"
              placeholder="tcp://localhost:1883"
              onChange={handleChange('jsonData.uri')}
            />
          </InlineField>
        </InlineFieldRow>
      </FieldSet>

      <FieldSet label="Authentication">
        <InlineFieldRow>
          <InlineField label="Username" labelWidth={10}>
            <Input
              width={30}
              name="username"
              value={username}
              autoComplete="off"
              onChange={handleChange('jsonData.username')}
            />
          </InlineField>
        </InlineFieldRow>
        <InlineFieldRow>
          <InlineField label="Password" labelWidth={10}>
            <SecretInput
              width={30}
              name="password"
              isConfigured={!!secureJsonFields.password}
              value={secureJsonData?.password ?? ''}
              onChange={onPasswordChange}
              onReset={onResetPassword}
            />
          </InlineField>
        </InlineFieldRow>
      </FieldSet>
    </>
  );
};
