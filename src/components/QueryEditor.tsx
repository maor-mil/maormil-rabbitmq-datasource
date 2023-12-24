import React, { useState, useEffect, useMemo } from 'react';

import { InlineField, InlineSwitch, FieldSet } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';

import { DataSource } from '../datasource';
import { JsonQueryModelComponent } from './JsonQueryModelComponent';
import { RabbitMQQuery, RabbitMQDataSourceOptions, JsonQueryModels } from '../types';
import { LABEL_WIDTH, SWITCH_WIDTH } from './consts';


type Props = QueryEditorProps<DataSource, RabbitMQQuery, RabbitMQDataSourceOptions>;

export const QueryEditor = (props: Props) => {
  const { query, onChange } = props;

  const [jsonQueryModels, setJsonQueryModels] = useState<JsonQueryModels>([]);

  useMemo(() => {
    if (!("areMessagesBase64Encrypted" in query)) {
      Object.assign(query, {areMessagesBase64Encrypted: false});
    }
  }, [query]);

  useEffect(() => {
    Object.assign(query, {jsonQueryModels: jsonQueryModels});
  }, [query, jsonQueryModels]);
  
  console.log(query)

  return (
    <>
      <InlineField label="Are messages Base64 encoded" labelWidth={LABEL_WIDTH} tooltip="Are the expected messages encoded in Base 64">
          <InlineSwitch
            onChange={(event) =>
              onChange({
                ...query,
                areMessagesBase64Encrypted: event!.currentTarget.checked
              })
            }
            value={query.areMessagesBase64Encrypted}
            width={SWITCH_WIDTH}
          />
      </InlineField>
      <br />
      <FieldSet label="Choose the JSON Schema">
        <JsonQueryModelComponent jsonQueryModels={jsonQueryModels} setJsonQueryModels={setJsonQueryModels}/>
      </FieldSet>
    </>
  );
}
