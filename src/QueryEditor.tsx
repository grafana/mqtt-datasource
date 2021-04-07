import React, { useState, useEffect } from 'react';
import { Form, Field, Input } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MqttDataSourceOptions, MqttQuery } from './types';
import { handleEvent } from 'handleEvent';

type Props = QueryEditorProps<DataSource, MqttQuery, MqttDataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const { query, onChange } = props;
  const [queryText, setQueryText] = useState(query.queryText ?? '');
  useEffect(() => onChange({ ...query, queryText }), [queryText]);

  return (
    <Form onSubmit={() => {}}>
      {() => (
        <Field label="Topic">
          <Input
            name="queryText"
            required
            value={queryText}
            css=""
            autoComplete="off"
            onChange={handleEvent(setQueryText)}
          />
        </Field>
      )}
    </Form>
  );
};
