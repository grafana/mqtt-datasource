import React, { useState, useEffect } from 'react';
import { Form, Field, FieldSet, Input } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MqttDataSourceOptions, MqttSecureJsonData } from './types';
import { handleEvent } from './handleEvent';

interface Props extends DataSourcePluginOptionsEditorProps<MqttDataSourceOptions> {}

export const ConfigEditor = (props: Props) => {
  const {
    options,
    options: { jsonData, secureJsonFields, secureJsonData },
    onOptionsChange,
  } = props;
  const secureData = (secureJsonData ?? {}) as MqttSecureJsonData;
  const [host, setHost] = useState(jsonData.host ?? 'localhost');
  const [port, setPort] = useState<number>(jsonData.port ?? 1883);
  const [username, setUsername] = useState(jsonData.username ?? '');
  const [password, setPassword] = useState(secureData.password ?? '');

  useEffect(() => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        host,
        port,
        username,
      },
      secureJsonData: {
        ...secureData,
        password,
      },
      secureJsonFields: {
        ...secureJsonFields,
        password: Boolean(password),
      },
    });
  }, [host, port, username, password]);

  return (
    <Form onSubmit={() => {}}>
      {() => (
        <>
          <FieldSet label="Connection">
            <Field label="Host">
              <Input name="host" required value={host} css="" autoComplete="off" onChange={handleEvent(setHost)} />
            </Field>
            <Field label="Port">
              <Input
                type="number"
                name="port"
                required
                value={port}
                css=""
                autoComplete="off"
                onChange={handleEvent(Number, setPort)}
              />
            </Field>
          </FieldSet>

          <FieldSet label="Authentication">
            <Field label="Username">
              <Input name="username" value={username} css="" autoComplete="off" onChange={handleEvent(setUsername)} />
            </Field>
            <Field label="Password">
              <Input
                type="password"
                name="password"
                css=""
                autoComplete="off"
                placeholder="************************"
                value={password}
                onChange={handleEvent(setPassword)}
              />
            </Field>
          </FieldSet>
        </>
      )}
    </Form>
  );
};
