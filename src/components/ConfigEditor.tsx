import React, { ChangeEvent, SetStateAction, useEffect } from 'react';

import { FieldSet, InlineField, InlineSwitch, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';

import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData, ExchangesOptions, BindingsOptions, StreamOptions } from '../types';
import { ExchangesComponent } from './ExchangesComponent';
import { BindingsComponent } from './BindingsComponent';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';
import { DEFAULT_OPTIONS, DEFAULT_PASSWORD, DEFAULT_STREAM_OPTIONS, hasMissingDefaults, withDefaults } from './defaults';

interface Props extends DataSourcePluginOptionsEditorProps<RabbitMQDataSourceOptions, RabbitMQSecureJsonData> {}

export const ConfigEditor = (props: Props) => {
  const { onOptionsChange, options } = props;
  const { secureJsonData, secureJsonFields } = options;

  // Always render (and write back) complete settings, even before the defaults are persisted.
  const jsonData = withDefaults(options.jsonData);
  const streamOptions = jsonData.streamOptions;

  // Every update must go through a single onOptionsChange call that carries the complete settings:
  // several calls in the same render all spread the same (stale) `options` and overwrite each other.
  const updateJsonData = (patch: Partial<RabbitMQDataSourceOptions>) => {
    onOptionsChange({ ...options, jsonData: { ...jsonData, ...patch } });
  };

  const updateStreamOptions = (patch: Partial<StreamOptions>) => {
    updateJsonData({ streamOptions: { ...streamOptions, ...patch } });
  };

  // Adapters with the React state-setter signature that the list components expect.
  const setExchanges = (action: SetStateAction<ExchangesOptions>) => {
    updateJsonData({ exchangesOptions: typeof action === 'function' ? action(jsonData.exchangesOptions) : action });
  };
  const setBindings = (action: SetStateAction<BindingsOptions>) => {
    updateJsonData({ bindingsOptions: typeof action === 'function' ? action(jsonData.bindingsOptions) : action });
  };

  const onNumericInputChange = (value: string, defaultValue: number, callback: (val: number) => void) => {
    const parsedValue = parseInt(value, 10);
    callback(isNaN(parsedValue) ? defaultValue : parsedValue);
  };

  // Secure field (only sent to the backend)
  const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData,
      secureJsonData: {
        password: event.target.value,
      },
    });
  };

  const onResetPassword = () => {
    onOptionsChange({
      ...options,
      jsonData,
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

  // Persist the defaults once when the editor opens. A brand-new data source (empty settings) also gets
  // the default password, as documented in the README.
  useEffect(() => {
    const isNewDataSource = Object.keys(options.jsonData ?? {}).length === 0;
    if (!isNewDataSource && !hasMissingDefaults(options.jsonData)) {
      return;
    }
    onOptionsChange({
      ...options,
      jsonData: withDefaults(options.jsonData),
      ...(isNewDataSource && {
        secureJsonFields: { ...options.secureJsonFields, password: true },
        secureJsonData: { ...options.secureJsonData, password: DEFAULT_PASSWORD },
      }),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div>
      <FieldSet label="Connection">
        <InlineField label="Host" labelWidth={LABEL_WIDTH} tooltip="Hostname (or the IP) of the RabbitMQ server">
          <Input
            onChange={(event) => updateJsonData({ host: event.currentTarget.value || DEFAULT_OPTIONS.host })}
            value={jsonData.host}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="AMQP Port" labelWidth={LABEL_WIDTH} tooltip="The AMQP port of the RabbitMQ server">
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.amqpPort, (value) =>
                updateJsonData({ amqpPort: value })
              )
            }
            value={jsonData.amqpPort.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Stream Port" labelWidth={LABEL_WIDTH} tooltip="The stream port of the RabbitMQ server">
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.streamPort, (value) =>
                updateJsonData({ streamPort: value })
              )
            }
            value={jsonData.streamPort.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="VHost" labelWidth={LABEL_WIDTH} tooltip="The virtual host the RabbitMQ server">
          <Input
            onChange={(event) => updateJsonData({ vHost: event.currentTarget.value || DEFAULT_OPTIONS.vHost })}
            value={jsonData.vHost}
            width={INPUT_WIDTH}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Authentication">
        <InlineField label="TLS Connection" labelWidth={LABEL_WIDTH} tooltip="Should use TLS to connect to the RabbitMQ server">
          <InlineSwitch
            onChange={(event) => updateJsonData({ tlsConnection: event.currentTarget.checked })}
            value={jsonData.tlsConnection ?? false}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField label="Username" labelWidth={LABEL_WIDTH} tooltip="Username to connect to the RabbitMQ server">
          <Input
            onChange={(event) => updateJsonData({ username: event.currentTarget.value || DEFAULT_OPTIONS.username })}
            value={jsonData.username}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Password" labelWidth={LABEL_WIDTH} tooltip="Password to connect to the RabbitMQ server">
          <SecretInput
            isConfigured={!!secureJsonFields.password}
            value={secureJsonData?.password ?? DEFAULT_PASSWORD}
            width={INPUT_WIDTH}
            onReset={onResetPassword}
            onChange={onPasswordChange}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Stream Settings">
        <InlineField label="Should Dispose Stream" labelWidth={LABEL_WIDTH} tooltip="Should delete this stream (in the RabbitMQ) when the RabbitMQ datasource is deleted">
          <InlineSwitch
            onChange={(event) => updateStreamOptions({ shouldDisposeStream: event.currentTarget.checked })}
            value={streamOptions.shouldDisposeStream}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField label="Stream Name" labelWidth={LABEL_WIDTH} tooltip="The stream name that will be created">
          <Input
            onChange={(event) =>
              updateStreamOptions({ streamName: event.currentTarget.value || DEFAULT_STREAM_OPTIONS.streamName })
            }
            value={streamOptions.streamName}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Consumer Name" labelWidth={LABEL_WIDTH} tooltip="The consumer name that will be created">
          <Input
            onChange={(event) => updateStreamOptions({ consumerName: event.currentTarget.value })}
            value={streamOptions.consumerName}
            placeholder="Consumer Name (can be empty)"
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Offset From Start" labelWidth={LABEL_WIDTH} tooltip="Should the consumer consume messages from the start or the end of the stored messages in the stream">
          <InlineSwitch
            onChange={(event) => updateStreamOptions({ offsetFromStart: event.currentTarget.checked })}
            value={streamOptions.offsetFromStart}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField label="Max Age" labelWidth={LABEL_WIDTH} tooltip="The max age of messages in the stream in nano-seconds (set to 0 to disable the max-age limit)">
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_STREAM_OPTIONS.maxAge, (value) =>
                updateStreamOptions({ maxAge: value })
              )
            }
            value={streamOptions.maxAge.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Max Length Bytes" labelWidth={LABEL_WIDTH} tooltip="The max length of messages in bytes in the stream (set to 0 to disable the max-length-bytes limit)">
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_STREAM_OPTIONS.maxLengthBytes, (value) =>
                updateStreamOptions({ maxLengthBytes: value })
              )
            }
            value={streamOptions.maxLengthBytes.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Max Segment Size Bytes" labelWidth={LABEL_WIDTH} tooltip="The max segment size in bytes in the stream">
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_STREAM_OPTIONS.maxSegmentSizeBytes, (value) =>
                updateStreamOptions({ maxSegmentSizeBytes: value })
              )
            }
            value={streamOptions.maxSegmentSizeBytes.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="CRC" labelWidth={LABEL_WIDTH} tooltip="When CRC control is disabled, the perfomance is increased">
          <InlineSwitch
            onChange={(event) => updateStreamOptions({ crc: event.currentTarget.checked })}
            value={streamOptions.crc}
            width={SWITCH_WIDTH}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Exchanges">
        <ExchangesComponent exchanges={jsonData.exchangesOptions} setExchanges={setExchanges} />
      </FieldSet>
      <FieldSet label="Bindings">
        <BindingsComponent bindings={jsonData.bindingsOptions} setBindings={setBindings} />
      </FieldSet>
      <FieldSet label="Advanced RabbitMQ Stream Settings">
        <InlineField label="Requested Heartbeat" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.requestedHeartbeat, (value) =>
                updateJsonData({ requestedHeartbeat: value })
              )
            }
            value={jsonData.requestedHeartbeat.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Requested Max FrameSize" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.requestedMaxFrameSize, (value) =>
                updateJsonData({ requestedMaxFrameSize: value })
              )
            }
            value={jsonData.requestedMaxFrameSize.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Write Buffer" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.writeBuffer, (value) =>
                updateJsonData({ writeBuffer: value })
              )
            }
            value={jsonData.writeBuffer.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Read Buffer" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onNumericInputChange(event.currentTarget.value, DEFAULT_OPTIONS.readBuffer, (value) =>
                updateJsonData({ readBuffer: value })
              )
            }
            value={jsonData.readBuffer.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="No Delay" labelWidth={LABEL_WIDTH}>
          <InlineSwitch
            onChange={(event) => updateJsonData({ noDelay: event.currentTarget.checked })}
            value={jsonData.noDelay}
            width={SWITCH_WIDTH}
          />
        </InlineField>
      </FieldSet>
    </div>
  );
};
