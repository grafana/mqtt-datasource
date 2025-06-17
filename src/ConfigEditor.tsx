import React, { SyntheticEvent } from 'react';

import {
  DataSourcePluginOptionsEditorProps,
  onUpdateDatasourceJsonDataOption,
  onUpdateDatasourceSecureJsonDataOption,
  updateDatasourcePluginJsonDataOption,
  updateDatasourcePluginResetOption,
} from '@grafana/data';
import { ConfigSection, DataSourceDescription } from '@grafana/plugin-ui';
import { Field, Input, SecretInput, Switch } from '@grafana/ui';
import { Divider } from './Divider';
import { TLSSecretsConfig } from './TLSConfig';
import { MqttDataSourceOptions, MqttSecureJsonData } from './types';

export const ConfigEditor = (props: DataSourcePluginOptionsEditorProps<MqttDataSourceOptions, MqttSecureJsonData>) => {
  const { options } = props;
  const jsonData = options.jsonData;

  const onResetPassword = () => {
    updateDatasourcePluginResetOption(props, 'password');
  };

  const onSwitchChanged = (property: keyof MqttDataSourceOptions) => {
    return (event: SyntheticEvent<HTMLInputElement>) => {
      updateDatasourcePluginJsonDataOption(props, property, event.currentTarget.checked);
    };
  };

  const WIDTH_LONG = 40;

  return (
    <>
      <DataSourceDescription
        dataSourceName="MQTT"
        docsLink="https://grafana.com/grafana/plugins/grafana-mqtt-datasource/?tab=overview"
        hasRequiredFields={true}
      />

      <Divider />

      <ConfigSection title="Connection">
        <Field label="URI" required>
          <Input
            width={WIDTH_LONG}
            name="URI"
            type="text"
            value={jsonData.uri || ''}
            onChange={onUpdateDatasourceJsonDataOption(props, 'uri')}
            placeholder="TCP (tcp://), TLS (tls://), or WebSocket (ws://)"
          />
        </Field>
      </ConfigSection>

      <Field label="Client ID" description="If not set, a random client ID is used.">
        <Input
          width={WIDTH_LONG}
          name="Client ID"
          type="text"
          value={jsonData.clientID || ''}
          onChange={onUpdateDatasourceJsonDataOption(props, 'clientID')}
        />
      </Field>

      <Divider />

      <ConfigSection title="Authentication">
        <Field label="Username">
          <Input
            width={WIDTH_LONG}
            value={jsonData.username || ''}
            placeholder="Username"
            onChange={onUpdateDatasourceJsonDataOption(props, 'username')}
          />
        </Field>

        <Field label="Password">
          <SecretInput
            width={WIDTH_LONG}
            placeholder="Password"
            isConfigured={options.secureJsonFields && options.secureJsonFields.password}
            onReset={onResetPassword}
            onBlur={onUpdateDatasourceSecureJsonDataOption(props, 'password')}
          />
        </Field>

        <Field
          label="Use TLS Client Auth"
          description="Enables TLS authentication using client cert configured in secure json data."
        >
          <Switch onChange={onSwitchChanged('tlsAuth')} value={jsonData.tlsAuth || false} />
        </Field>

        <Field
          label="Skip TLS Verification"
          description="When enabled, skips verification of the MQTT server's TLS certificate chain and host name."
        >
          <Switch onChange={onSwitchChanged('tlsSkipVerify')} value={jsonData.tlsSkipVerify || false} />
        </Field>

        <Field label="With CA Cert" description="Needed for verifying servers with self-signed TLS Certs.">
          <Switch onChange={onSwitchChanged('tlsAuthWithCACert')} value={jsonData.tlsAuthWithCACert || false} />
        </Field>
      </ConfigSection>

      {jsonData.tlsAuth || jsonData.tlsAuthWithCACert ? (
        <>
          <Divider />

          <ConfigSection title="TLS Configuration">
            {jsonData.tlsAuth || jsonData.tlsAuthWithCACert ? (
              <TLSSecretsConfig
                showCACert={jsonData.tlsAuthWithCACert}
                showKeyPair={jsonData.tlsAuth}
                editorProps={props}
                labelWidth={WIDTH_LONG}
              />
            ) : null}
          </ConfigSection>
        </>
      ) : null}
    </>
  );
};
