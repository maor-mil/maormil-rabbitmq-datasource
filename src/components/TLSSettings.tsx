import React from 'react';

import { FieldSet, InlineField, InlineSwitch, Input, RadioButtonGroup, SecretTextArea } from '@grafana/ui';

import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData, TlsMinVersion } from '../types';
import { INPUT_WIDTH, LABEL_WIDTH, SWITCH_WIDTH } from './consts';

export type TLSSecretKey = keyof Pick<RabbitMQSecureJsonData, 'tlsCACert' | 'tlsClientCert' | 'tlsClientKey'>;

interface Props {
  jsonData: RabbitMQDataSourceOptions;
  secureJsonData?: RabbitMQSecureJsonData;
  secureJsonFields: Record<string, boolean>;
  updateJsonData: (patch: Partial<RabbitMQDataSourceOptions>) => void;
  onSecretChange: (key: TLSSecretKey, value: string) => void;
  onSecretReset: (key: TLSSecretKey) => void;
}

const TLS_VERSIONS: Array<{ label: string; value: TlsMinVersion }> = [
  { label: 'TLS 1.2', value: '1.2' },
  { label: 'TLS 1.3', value: '1.3' },
];

const PEM_ROWS = 7;

/** TLS options of the AMQP and stream connections. Rendered only when "TLS Connection" is on. */
export function TLSSettings({
  jsonData,
  secureJsonData,
  secureJsonFields,
  updateJsonData,
  onSecretChange,
  onSecretReset,
}: Props) {
  const secret = (key: TLSSecretKey, label: string, placeholder: string, tooltip: string) => (
    <InlineField label={label} labelWidth={LABEL_WIDTH} tooltip={tooltip} grow>
      <SecretTextArea
        aria-label={label}
        isConfigured={!!secureJsonFields[key]}
        value={secureJsonData?.[key] ?? ''}
        placeholder={placeholder}
        rows={PEM_ROWS}
        cols={INPUT_WIDTH * 2}
        onChange={(event) => onSecretChange(key, event.currentTarget.value)}
        onReset={() => onSecretReset(key)}
      />
    </InlineField>
  );

  return (
    <FieldSet label="TLS Settings">
      <InlineField
        label="Skip TLS Verify"
        labelWidth={LABEL_WIDTH}
        tooltip="Do not verify the RabbitMQ server certificate. Insecure - only use it for testing."
      >
        <InlineSwitch
          onChange={(event) => updateJsonData({ tlsSkipVerify: event.currentTarget.checked })}
          value={jsonData.tlsSkipVerify}
          width={SWITCH_WIDTH}
        />
      </InlineField>
      <InlineField
        label="Server Name"
        labelWidth={LABEL_WIDTH}
        tooltip="Name used to verify the server certificate (SNI). Leave empty to use the host."
      >
        <Input
          onChange={(event) => updateJsonData({ serverName: event.currentTarget.value })}
          value={jsonData.serverName}
          placeholder={jsonData.host}
          width={INPUT_WIDTH}
        />
      </InlineField>
      <InlineField
        label="Min TLS Version"
        labelWidth={LABEL_WIDTH}
        tooltip="Lowest TLS version accepted for the connection"
      >
        <RadioButtonGroup
          options={TLS_VERSIONS}
          value={jsonData.tlsMinVersion}
          onChange={(value) => updateJsonData({ tlsMinVersion: value })}
        />
      </InlineField>
      <InlineField
        label="With CA Cert"
        labelWidth={LABEL_WIDTH}
        tooltip="Verify the server certificate with your own CA (e.g. self-signed) instead of the system root CAs"
      >
        <InlineSwitch
          onChange={(event) => updateJsonData({ tlsAuthWithCACert: event.currentTarget.checked })}
          value={jsonData.tlsAuthWithCACert}
          width={SWITCH_WIDTH}
        />
      </InlineField>
      {jsonData.tlsAuthWithCACert &&
        secret('tlsCACert', 'CA Cert', '-----BEGIN CERTIFICATE-----', 'PEM encoded CA certificate(s)')}
      <InlineField
        label="TLS Client Auth"
        labelWidth={LABEL_WIDTH}
        tooltip="Present a client certificate (mutual TLS). Required for the EXTERNAL authentication mechanism."
      >
        <InlineSwitch
          onChange={(event) => updateJsonData({ tlsAuth: event.currentTarget.checked })}
          value={jsonData.tlsAuth}
          width={SWITCH_WIDTH}
        />
      </InlineField>
      {jsonData.tlsAuth && (
        <>
          {secret('tlsClientCert', 'Client Cert', '-----BEGIN CERTIFICATE-----', 'PEM encoded client certificate')}
          {secret(
            'tlsClientKey',
            'Client Key',
            '-----BEGIN PRIVATE KEY-----',
            'PEM encoded private key of the client certificate'
          )}
        </>
      )}
    </FieldSet>
  );
}
