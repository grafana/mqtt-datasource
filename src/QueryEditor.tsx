import React from 'react';
import { Input, InlineFieldRow, InlineField } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MqttDataSourceOptions, MqttQuery } from './types';

type Props = QueryEditorProps<DataSource, MqttQuery, MqttDataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const { query, onChange, onRunQuery } = props;

  return (
    <>
      <InlineFieldRow>
        <InlineField label="Topic" labelWidth={8} grow>
          <Input
            name="topic"
            required
            placeholder='e.g. "home/bedroom/temperature"'
            value={query.topic}
            onBlur={onRunQuery}
            onChange={(e) => onChange({...query, topic: e.currentTarget.value })}
          />
        </InlineField>
      </InlineFieldRow>
    </>
  );
};
