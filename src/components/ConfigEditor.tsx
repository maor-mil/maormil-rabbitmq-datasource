import React, { ChangeEvent, SetStateAction } from 'react';

import { FieldSet, InlineField, InlineSwitch, Input, RadioButtonGroup, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';

import {
  RabbitMQDataSourceOptions,
  RabbitMQSecureJsonData,
  ExchangesOptions,
  BindingsOptions,
  SaslMechanism,
  StreamOptions,
} from '../types';
import { ExchangesComponent } from './ExchangesComponent';
import { BindingsComponent } from './BindingsComponent';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';
import {
  DEFAULT_OPTIONS,
  DEFAULT_PASSWORD,
  DEFAULT_STREAM_OPTIONS,
  PLAIN_PORTS,
  TLS_PORTS,
  withDefaults,
} from './defaults';
import { TLSSecretKey, TLSSettings } from './TLSSettings';

const SASL_MECHANISMS: Array<{ label: string; value: SaslMechanism; description: string }> = [
  { label: 'Username & password', value: 'PLAIN', description: 'SASL PLAIN' },
  {
    label: 'Client certificate',
    value: 'EXTERNAL',
    description: 'SASL EXTERNAL: the TLS client certificate identifies the user (requires TLS Client Auth)',
  },
];

interface Props extends DataSourcePluginOptionsEditorProps<RabbitMQDataSourceOptions, RabbitMQSecureJsonData> {}

export const ConfigEditor = (props: Props) => {
  const { onOptionsChange, options } = props;
  const { secureJsonData, secureJsonFields } = options;

  // Always render (and write back) complete settings. Defaults are NOT persisted on mount: Grafana 13 discards
  // onOptionsChange calls made while its settings page is still initialising. The backend applies the same
  // defaults to any missing value (pkg/rabbitmqclient/defaults.go), including the default password.
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

  // Secure fields (only sent to the backend). Keep the other, not yet saved secrets when one changes.
  const onSecretChange = (key: keyof RabbitMQSecureJsonData, value: string) => {
    onOptionsChange({ ...options, jsonData, secureJsonData: { ...options.secureJsonData, [key]: value } });
  };

  const onSecretReset = (key: keyof RabbitMQSecureJsonData) => {
    onOptionsChange({
      ...options,
      jsonData,
      secureJsonFields: { ...options.secureJsonFields, [key]: false },
      secureJsonData: { ...options.secureJsonData, [key]: '' },
    });
  };

  const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => onSecretChange('password', event.target.value);
  const onResetPassword = () => onSecretReset('password');

  // Switch between the standard plain and TLS ports, unless the user configured custom ports.
  const onTLSConnectionChange = (enabled: boolean) => {
    const [from, to] = enabled ? [PLAIN_PORTS, TLS_PORTS] : [TLS_PORTS, PLAIN_PORTS];
    updateJsonData({
      tlsConnection: enabled,
      amqpPort: jsonData.amqpPort === from.amqpPort ? to.amqpPort : jsonData.amqpPort,
      streamPort: jsonData.streamPort === from.streamPort ? to.streamPort : jsonData.streamPort,
    });
  };

  const usesPassword = jsonData.saslMechanism !== 'EXTERNAL';

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
        <InlineField
          label="AMQP Port"
          labelWidth={LABEL_WIDTH}
          tooltip="The AMQP port of the RabbitMQ server (5672, or 5671 with TLS)"
        >
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
        <InlineField
          label="Stream Port"
          labelWidth={LABEL_WIDTH}
          tooltip="The stream port of the RabbitMQ server (5552, or 5551 with TLS)"
        >
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
        <InlineField
          label="TLS Connection"
          labelWidth={LABEL_WIDTH}
          tooltip="Should use TLS to connect to the RabbitMQ server"
        >
          <InlineSwitch
            onChange={(event) => onTLSConnectionChange(event.currentTarget.checked)}
            value={jsonData.tlsConnection ?? false}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField
          label="Authentication Mechanism"
          labelWidth={LABEL_WIDTH}
          tooltip="Username & password (SASL PLAIN), or the TLS client certificate (SASL EXTERNAL; the broker needs the rabbitmq_auth_mechanism_ssl plugin)"
        >
          <RadioButtonGroup
            options={SASL_MECHANISMS}
            value={jsonData.saslMechanism}
            onChange={(value) => updateJsonData({ saslMechanism: value })}
          />
        </InlineField>
        {usesPassword && (
          <>
            <InlineField label="Username" labelWidth={LABEL_WIDTH} tooltip="Username to connect to the RabbitMQ server">
              <Input
                onChange={(event) =>
                  updateJsonData({ username: event.currentTarget.value || DEFAULT_OPTIONS.username })
                }
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
          </>
        )}
      </FieldSet>
      {jsonData.tlsConnection && (
        <TLSSettings
          jsonData={jsonData}
          secureJsonData={secureJsonData}
          secureJsonFields={secureJsonFields}
          updateJsonData={updateJsonData}
          onSecretChange={(key: TLSSecretKey, value: string) => onSecretChange(key, value)}
          onSecretReset={(key: TLSSecretKey) => onSecretReset(key)}
        />
      )}
      <FieldSet label="Stream Settings">
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
        <InlineField
          label="Offset From Start"
          labelWidth={LABEL_WIDTH}
          tooltip="Should the consumer consume messages from the start or the end of the stored messages in the stream"
        >
          <InlineSwitch
            onChange={(event) => updateStreamOptions({ offsetFromStart: event.currentTarget.checked })}
            value={streamOptions.offsetFromStart}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField
          label="Max Age"
          labelWidth={LABEL_WIDTH}
          tooltip="The max age of messages in the stream in nano-seconds (set to 0 to disable the max-age limit)"
        >
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
        <InlineField
          label="Max Length Bytes"
          labelWidth={LABEL_WIDTH}
          tooltip="The max length of messages in bytes in the stream (set to 0 to disable the max-length-bytes limit)"
        >
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
        <InlineField
          label="Max Segment Size Bytes"
          labelWidth={LABEL_WIDTH}
          tooltip="The max segment size in bytes in the stream"
        >
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
        <InlineField
          label="CRC"
          labelWidth={LABEL_WIDTH}
          tooltip="When CRC control is disabled, the perfomance is increased"
        >
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
