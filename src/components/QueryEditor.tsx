import React from 'react';

import { QueryEditorProps } from '@grafana/data';
import {
  Alert,
  InlineField,
  InlineFieldRow,
  InlineSwitch,
  Input,
  RadioButtonGroup,
  Stack,
  TagsInput,
  TextArea,
} from '@grafana/ui';

import type { DataSource } from '../datasource';
import { QueueMode, RabbitMQDataSourceOptions, RabbitMQQuery, SourceType, StreamOffset } from '../types';

type Props = QueryEditorProps<DataSource, RabbitMQQuery, RabbitMQDataSourceOptions>;

const LABEL_WIDTH = 16;
const INPUT_WIDTH = 32;

const SOURCES: Array<{ label: string; value: SourceType; description: string }> = [
  { label: 'Data source stream', value: 'datasource', description: 'The stream configured on the data source' },
  { label: 'Stream', value: 'stream', description: 'Any existing stream (stream protocol, RabbitMQ 3.9+)' },
  { label: 'Queue', value: 'queue', description: 'Classic or quorum queues over AMQP 1.0 (RabbitMQ 4.0+)' },
];

const OFFSETS: Array<{ label: string; value: StreamOffset; description: string }> = [
  { label: 'First', value: 'first', description: 'Replay every message still stored in the stream' },
  { label: 'Last', value: 'last', description: 'Start from the last chunk of messages' },
  { label: 'Next', value: 'next', description: 'Only messages published from now on' },
];

const QUEUE_MODES: Array<{ label: string; value: QueueMode; description: string }> = [
  {
    label: 'Tap an exchange',
    value: 'tap',
    description: 'A temporary queue bound to the exchange receives a copy of every routed message',
  },
  {
    label: 'Consume a queue',
    value: 'existing',
    description: 'Read an existing queue - removes the messages from it',
  },
];

/** Chooses what the panel streams. Every change starts streaming the new source. */
export const QueryEditor = ({ query, onChange, onRunQuery }: Props) => {
  const source = query.source ?? 'datasource';
  const queueMode = query.queueMode ?? 'tap';

  // Radio buttons apply immediately; text fields apply on blur.
  const update = (patch: Partial<RabbitMQQuery>, run = true) => {
    onChange({ ...query, ...patch });
    if (run) {
      onRunQuery();
    }
  };
  const text = (
    field: 'stream' | 'exchange' | 'routingKey' | 'queue',
    label: string,
    tooltip: string,
    placeholder = ''
  ) => (
    <InlineField label={label} labelWidth={LABEL_WIDTH} tooltip={tooltip}>
      <Input
        width={INPUT_WIDTH}
        value={query[field] ?? ''}
        placeholder={placeholder}
        onChange={(event) => update({ [field]: event.currentTarget.value }, false)}
        onBlur={() => onRunQuery()}
      />
    </InlineField>
  );

  return (
    <Stack direction="column" gap={0}>
      <InlineFieldRow>
        <InlineField label="Source" labelWidth={LABEL_WIDTH} tooltip="What this panel streams">
          <RadioButtonGroup options={SOURCES} value={source} onChange={(value) => update({ source: value })} />
        </InlineField>
      </InlineFieldRow>

      {source === 'stream' && (
        <>
          <InlineFieldRow>
            {text('stream', 'Stream', 'Name of an existing stream (the plugin does not create it)')}
            <InlineField label="Offset" labelWidth={LABEL_WIDTH} tooltip="Where to start reading the stream">
              <RadioButtonGroup
                options={OFFSETS}
                value={query.offset ?? 'first'}
                onChange={(value) => update({ offset: value })}
              />
            </InlineField>
          </InlineFieldRow>
          <InlineFieldRow>
            <InlineField
              label="Filter values"
              labelWidth={LABEL_WIDTH}
              tooltip="Stream filtering (RabbitMQ 3.13+): only chunks containing these filter values are read. Publishers set the value per message (e.g. the x-stream-filter-value header)."
            >
              <TagsInput
                id={`filter-values-${query.refId}`}
                width={INPUT_WIDTH}
                tags={query.filterValues ?? []}
                placeholder="Add a value and press Enter"
                addOnBlur
                onChange={(tags) => update({ filterValues: tags })}
              />
            </InlineField>
            {(query.filterValues?.length ?? 0) > 0 && (
              <InlineField
                label="Match unfiltered"
                labelWidth={LABEL_WIDTH}
                tooltip="Also read messages that were published without a filter value"
              >
                <InlineSwitch
                  value={query.matchUnfiltered ?? false}
                  onChange={(event) => update({ matchUnfiltered: event.currentTarget.checked })}
                />
              </InlineField>
            )}
          </InlineFieldRow>
          <InlineFieldRow>
            <InlineField
              label="SQL filter"
              labelWidth={LABEL_WIDTH}
              grow
              tooltip="SQL filter expression that RabbitMQ evaluates for every message (RabbitMQ 4.2+). Can use message properties (p.subject, p.creation_time, ...) and application properties."
            >
              <TextArea
                aria-label="SQL filter"
                rows={2}
                value={query.sql ?? ''}
                placeholder="region = 'EMEA' AND p.subject LIKE 'order.%'"
                onChange={(event) => update({ sql: event.currentTarget.value }, false)}
                onBlur={() => onRunQuery()}
              />
            </InlineField>
          </InlineFieldRow>
          {query.sql && (
            <Alert severity="info" title="SQL filters are read over AMQP 1.0">
              A stream with an SQL filter is read over AMQP 1.0 on the AMQP port (RabbitMQ 4.2 or later). An invalid
              expression is reported on the panel.{' '}
              <a
                href="https://www.rabbitmq.com/docs/stream-filtering#sql-filter-expressions"
                target="_blank"
                rel="noreferrer"
              >
                Syntax
              </a>
            </Alert>
          )}
        </>
      )}

      {source === 'queue' && (
        <>
          <InlineFieldRow>
            <InlineField label="Mode" labelWidth={LABEL_WIDTH} tooltip="How the queue is read">
              <RadioButtonGroup
                options={QUEUE_MODES}
                value={queueMode}
                onChange={(value) => update({ queueMode: value })}
              />
            </InlineField>
          </InlineFieldRow>
          {queueMode === 'tap' ? (
            <InlineFieldRow>
              {text('exchange', 'Exchange', 'Exchange whose messages the panel shows (it must exist)')}
              {text(
                'routingKey',
                'Routing key',
                'Binding key of the temporary queue, e.g. "orders.#" for a topic exchange; ignored by fanout exchanges',
                '#'
              )}
            </InlineFieldRow>
          ) : (
            <>
              <InlineFieldRow>{text('queue', 'Queue', 'Name of an existing classic or quorum queue')}</InlineFieldRow>
              <Alert severity="warning" title="Consuming a queue removes its messages">
                The panel competes with the other consumers of this queue, and every message it reads is acknowledged
                and removed. Use &quot;Tap an exchange&quot; to watch traffic without affecting applications.
              </Alert>
            </>
          )}
          <Alert severity="info" title="Requires RabbitMQ 4.0 or later">
            Queues are read over AMQP 1.0 on the AMQP port, with the data source&apos;s credentials and TLS settings.
          </Alert>
        </>
      )}
    </Stack>
  );
};
