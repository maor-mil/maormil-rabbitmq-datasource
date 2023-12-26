import React, { ChangeEvent, useState, useMemo, useEffect } from 'react';

import { FieldSet, InlineField, InlineSwitch, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';

import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData, Exchanges, Bindings, StreamOptions } from '../types';
import { ExchangesComponent } from './ExchangesComponent';
import { BindingsComponent } from './BindingsComponent';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';

interface Props extends DataSourcePluginOptionsEditorProps<RabbitMQDataSourceOptions, RabbitMQSecureJsonData> {}

export const ConfigEditor = (props: Props) => {
  const {
    onOptionsChange,
    options,
    options: { jsonData, secureJsonData, secureJsonFields },
  } = props;

  const [streamOptions, setStreamOptions] = useState<StreamOptions>({
    streamName: "",
    maxLengthBytes: 2000000000,
    maxAge: 1,
    maxSegmentSizeBytes: 500000000,
    consumerName: "",
    crc: false
  });
  const [exchanges, setExchanges] = useState<Exchanges>([]);
  const [bindings, setBindings] = useState<Bindings>([]);

  // Secure field (only sent to the backend)
  const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        password: event.target.value,
      },
    });
  };

  const onResetPassword = () => {
    onOptionsChange({
      ...options,
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

  useMemo(() => {
    if (!("host" in jsonData)) {
      Object.assign(jsonData, {host: "localhost"});
    }
    if (!("amqpPort" in jsonData)) {
      Object.assign(jsonData, {amqpPort: 5672});
    }
    if (!("streamPort" in jsonData)) {
      Object.assign(jsonData, {streamPort: 5552});
    }
    if (!("vHost" in jsonData)) {
      Object.assign(jsonData, {vHost: "/"});
    }
    if (!("tlsConnection" in jsonData)) {
      Object.assign(jsonData, {tlsConnection: false});
    }
    if (!("username" in jsonData)) {
      Object.assign(jsonData, {username: "guest"});
    }
    if (!("streamOptions" in jsonData)) {
      Object.assign(jsonData, {streamOptions: streamOptions});
    }
    if (!("requestedHeartbeat" in jsonData)) {
      Object.assign(jsonData, {requestedHeartbeat: 60});
    }
    if (!("requestedMaxFrameSize" in jsonData)) {
      Object.assign(jsonData, {requestedMaxFrameSize: 1048576});
    }
    if (!("writeBuffer" in jsonData)) {
      Object.assign(jsonData, {writeBuffer: 8192});
    }
    if (!("readBuffer" in jsonData)) {
      Object.assign(jsonData, {readBuffer: 65536});
    }
    if (!("noDelay" in jsonData)) {
      Object.assign(jsonData, {noDelay: false});
    }
    if (!("exchangesOptions" in jsonData)) {
      Object.assign(jsonData, {exchangesOptions: exchanges});
    }
    if (!("bindingsOptions" in jsonData)) {
      Object.assign(jsonData, {bindingsOptions: bindings});
    }
  }, [jsonData, bindings, exchanges, streamOptions]);

  useEffect(() => {
    Object.assign(jsonData, {exchangesOptions: exchanges});
  }, [jsonData, exchanges]);
  useEffect(() => {
    Object.assign(jsonData, {bindingsOptions: bindings});
  }, [jsonData, bindings]);
  useEffect(() => {
    Object.assign(jsonData, {streamOptions: streamOptions});
  }, [jsonData, streamOptions]);

  const rabbitmqDefaultPassword = "guest";
  console.log(jsonData);

  return (
    <div className="gf-form-group">
      <FieldSet label="Connection">
        <InlineField label="Host" labelWidth={LABEL_WIDTH} tooltip="Host of the RabbitMQ server">
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, host: event.currentTarget.value },
              })
            }
            placeholder={jsonData.host}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="AmqpPort" labelWidth={LABEL_WIDTH} tooltip="The AMQP port of the RabbitMQ server">
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, amqpPort: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.amqpPort.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="StreamPort" labelWidth={LABEL_WIDTH} tooltip="The stream port of the RabbitMQ server">
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, streamPort: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.streamPort.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="VHost" labelWidth={LABEL_WIDTH} tooltip="The VHost the RabbitMQ server">
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, vHost: event.currentTarget.value },
              })
            }
            placeholder={jsonData.vHost.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Authentication">
        <InlineField label="TlsConnection" labelWidth={LABEL_WIDTH} tooltip="Should use TLS to connect to the RabbitMQ server">
          <InlineSwitch
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, tlsConnection: event!.currentTarget.checked },
              })
            }
            value={jsonData.tlsConnection}
            width={SWITCH_WIDTH}
          />
        </InlineField>
        <InlineField label="Username" labelWidth={LABEL_WIDTH} tooltip="Username to connect to the RabbitMQ server">
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, username: event.currentTarget.value },
              })
            }
            placeholder={jsonData.username}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Password" labelWidth={LABEL_WIDTH} tooltip="Password to connect to the RabbitMQ server">
          <SecretInput
            isConfigured={!!secureJsonFields.password}
            value={secureJsonData?.password ?? rabbitmqDefaultPassword}
            placeholder={rabbitmqDefaultPassword}
            width={INPUT_WIDTH}
            onReset={onResetPassword}
            onChange={onPasswordChange}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Stream Settings">
        <InlineField label="Stream Name" labelWidth={LABEL_WIDTH} tooltip="The stream name that will be created">
          <Input
            onChange={(event) =>
              setStreamOptions({
                ...streamOptions,
                streamName: event.currentTarget.value,
              })
            }
            placeholder={streamOptions.streamName}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Consumer Name" labelWidth={LABEL_WIDTH} tooltip="The consumer name that will be created">
          <Input
            onChange={(event) =>
              setStreamOptions({
                ...streamOptions,
                consumerName: event.currentTarget.value,
              })
            }
            placeholder={streamOptions.consumerName}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Max Length Bytes" labelWidth={LABEL_WIDTH} tooltip="The max length of messages (in bytes) in the stream">
          <Input
            onChange={(event) =>
              setStreamOptions({
                ...streamOptions,
                maxLengthBytes: parseInt(event.currentTarget.value, 10),
              })
            }
            placeholder={streamOptions.maxLengthBytes.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Max Age" labelWidth={LABEL_WIDTH} tooltip="The max age of messages in the stream">
          <Input
            onChange={(event) =>
              setStreamOptions({
                ...streamOptions,
                maxAge: parseInt(event.currentTarget.value, 10),
              })
            }
            placeholder={streamOptions.maxAge.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="CRC" labelWidth={LABEL_WIDTH} tooltip="When crc control is disabed, the perfomance is increased">
          <InlineSwitch
            onChange={(event) =>
              setStreamOptions({
                ...streamOptions,
                crc: event!.currentTarget.checked,
              })
            }
            value={streamOptions.crc}
            width={SWITCH_WIDTH}
          />
        </InlineField>
      </FieldSet>
      <FieldSet label="Exchanges (Create new exchanges in the RabbitMQ)">
        <ExchangesComponent exchanges={exchanges} setExchanges={setExchanges}/>
      </FieldSet>
      <FieldSet label="Bindings (Create new bindings in the RabbitMQ)">
        <BindingsComponent bindings={bindings} setBindings={setBindings}/>
      </FieldSet>
      <FieldSet label="Advanced RabbitMQ Stream Settings (Change these only if you really know what you are doing)">
        <InlineField label="Requested Heartbeat" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, requestedHeartbeat: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.requestedHeartbeat.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Requested Max FrameSize" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, requestedMaxFrameSize: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.requestedMaxFrameSize.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Write Buffer" labelWidth={LABEL_WIDTH}>
          <Input
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, writeBuffer: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.writeBuffer.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="Read Buffer" labelWidth={LABEL_WIDTH}>
          <Input
             onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, readBuffer: parseInt(event.currentTarget.value, 10) },
              })
            }
            placeholder={jsonData.readBuffer.toString()}
            width={INPUT_WIDTH}
          />
        </InlineField>
        <InlineField label="No Delay" labelWidth={LABEL_WIDTH}>
          <InlineSwitch
            onChange={(event) =>
              onOptionsChange({
                ...options,
                jsonData: { ...options.jsonData, noDelay: event!.currentTarget.checked },
              })
            }
            value={jsonData.noDelay}
            width={SWITCH_WIDTH}
          />
        </InlineField>
      </FieldSet>
    </div>
  );
}
