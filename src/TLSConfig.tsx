import React from 'react';

import {
  DataSourceJsonData,
  DataSourcePluginOptionsEditorProps,
  KeyValue,
  onUpdateDatasourceSecureJsonDataOption,
  updateDatasourcePluginResetOption,
} from '@grafana/data';
import { Field, Icon, Label, SecretTextArea, Stack, Tooltip } from '@grafana/ui';

export interface Props<T extends DataSourceJsonData, S> {
  editorProps: DataSourcePluginOptionsEditorProps<T, S>;
  showCACert?: boolean;
  showKeyPair?: boolean;
  secureJsonFields?: KeyValue<Boolean>;
  labelWidth?: number;
}

export const TLSSecretsConfig = <T extends DataSourceJsonData, S extends {} = {}>(props: Props<T, S>) => {
  const { editorProps, showCACert, showKeyPair = true } = props;
  const { secureJsonFields } = editorProps.options;
  return (
    <>
      {showCACert ? (
        <Field
          label={
            <Label>
              <Stack gap={0.5}>
                <span>TLS CA Certificate</span>
                <Tooltip
                  content={
                    <span>
                      If a Certificate Authority certificate is required to verify the server&#39;s certificate, provide
                      it here.
                    </span>
                  }
                >
                  <Icon name="info-circle" size="sm" />
                </Tooltip>
              </Stack>
            </Label>
          }
        >
          <SecretTextArea
            placeholder="-----BEGIN CERTIFICATE-----"
            cols={45}
            rows={7}
            isConfigured={secureJsonFields && secureJsonFields.tlsCACert}
            onChange={onUpdateDatasourceSecureJsonDataOption(editorProps, 'tlsCACert')}
            onReset={() => {
              updateDatasourcePluginResetOption(editorProps, 'tlsCACert');
            }}
          />
        </Field>
      ) : null}
      {showKeyPair ? (
        <Field
          label={
            <Label>
              <Stack gap={0.5}>
                <span>TLS Client Certificate</span>
                <Tooltip
                  content={
                    <span>To authenticate with an TLS client certificate, provide the client certificate here.</span>
                  }
                >
                  <Icon name="info-circle" size="sm" />
                </Tooltip>
              </Stack>
            </Label>
          }
        >
          <SecretTextArea
            placeholder="-----BEGIN CERTIFICATE-----"
            cols={45}
            rows={7}
            isConfigured={secureJsonFields && secureJsonFields.tlsClientCert}
            onChange={onUpdateDatasourceSecureJsonDataOption(editorProps, 'tlsClientCert')}
            onReset={() => {
              updateDatasourcePluginResetOption(editorProps, 'tlsClientCert');
            }}
          />
        </Field>
      ) : null}
      {showKeyPair ? (
        <Field
          label={
            <Label>
              <Stack gap={0.5}>
                <span>TLS Client Key</span>
                <Tooltip
                  content={<span>To authenticate with a client TLS certificate, provide the private key here.</span>}
                >
                  <Icon name="info-circle" size="sm" />
                </Tooltip>
              </Stack>
            </Label>
          }
        >
          <SecretTextArea
            placeholder="-----BEGIN RSA PRIVATE KEY-----"
            cols={45}
            rows={7}
            isConfigured={secureJsonFields && secureJsonFields.tlsClientKey}
            onChange={onUpdateDatasourceSecureJsonDataOption(editorProps, 'tlsClientKey')}
            onReset={() => {
              updateDatasourcePluginResetOption(editorProps, 'tlsClientKey');
            }}
          />
        </Field>
      ) : null}
    </>
  );
};
