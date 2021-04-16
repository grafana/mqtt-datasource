import React from 'react';
import { Form, Field, Input } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MqttDataSourceOptions, MqttQuery } from './types';
import { handlerFactory } from 'handleEvent';

type Props = QueryEditorProps<DataSource, MqttQuery, MqttDataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const { query, onChange } = props;
  const handleEvent = handlerFactory(query, onChange);

  return (
    <Form onSubmit={() => {}}>
      {() => (
        <Field label="Topic">
          <Input
            name="queryText"
            required
            value={query.queryText}
            css=""
            autoComplete="off"
            onChange={handleEvent('queryText')}
          />
        </Field>
      )}
    </Form>
  );
};
