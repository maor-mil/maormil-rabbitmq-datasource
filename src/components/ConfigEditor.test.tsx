import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataSourceSettings } from '@grafana/data';

import { ConfigEditor } from './ConfigEditor';
import { DEFAULT_OPTIONS, DEFAULT_PASSWORD, hasMissingDefaults, withDefaults } from './defaults';
import { RabbitMQDataSourceOptions, RabbitMQSecureJsonData } from '../types';

type Settings = DataSourceSettings<RabbitMQDataSourceOptions, RabbitMQSecureJsonData>;

const settings = (jsonData: Partial<RabbitMQDataSourceOptions>, secureJsonFields = {}): Settings =>
  ({
    id: 1,
    uid: 'test',
    orgId: 1,
    name: 'RabbitMQ',
    type: 'maormil-rabbitmq-datasource',
    jsonData,
    secureJsonFields,
    readOnly: false,
  }) as unknown as Settings;

describe('ConfigEditor', () => {
  it('persists every default and the default password in a single update for a new data source', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={settings({})} onOptionsChange={onOptionsChange} />);

    expect(onOptionsChange).toHaveBeenCalledTimes(1);
    const updated: Settings = onOptionsChange.mock.calls[0][0];
    expect(updated.jsonData).toEqual(DEFAULT_OPTIONS);
    expect(updated.secureJsonData?.password).toBe(DEFAULT_PASSWORD);
  });

  it('does not touch a fully configured data source on open', () => {
    const onOptionsChange = jest.fn();
    render(
      <ConfigEditor
        options={settings({ ...DEFAULT_OPTIONS, host: 'broker' }, { password: true })}
        onOptionsChange={onOptionsChange}
      />
    );
    expect(onOptionsChange).not.toHaveBeenCalled();
  });

  it('keeps all other settings when a single field changes', () => {
    const onOptionsChange = jest.fn();
    render(
      <ConfigEditor
        options={settings({ ...DEFAULT_OPTIONS, host: 'broker' }, { password: true })}
        onOptionsChange={onOptionsChange}
      />
    );

    fireEvent.change(screen.getByRole('textbox', { name: 'Stream Name' }), { target: { value: 'my.stream' } });

    const updated: Settings = onOptionsChange.mock.calls[0][0];
    expect(updated.jsonData).toEqual({
      ...DEFAULT_OPTIONS,
      host: 'broker',
      streamOptions: { ...DEFAULT_OPTIONS.streamOptions, streamName: 'my.stream' },
    });
  });
});

describe('defaults', () => {
  it('fills missing top-level and nested stream values without overriding existing ones', () => {
    const filled = withDefaults({ host: 'broker', streamOptions: { streamName: 'custom' } as any });
    expect(filled.host).toBe('broker');
    expect(filled.amqpPort).toBe(DEFAULT_OPTIONS.amqpPort);
    expect(filled.streamOptions.streamName).toBe('custom');
    expect(filled.streamOptions.maxAge).toBe(DEFAULT_OPTIONS.streamOptions.maxAge);
  });

  it('detects missing values', () => {
    expect(hasMissingDefaults({})).toBe(true);
    expect(hasMissingDefaults({ ...DEFAULT_OPTIONS, streamOptions: { streamName: 'x' } as any })).toBe(true);
    expect(hasMissingDefaults(DEFAULT_OPTIONS)).toBe(false);
  });
});
