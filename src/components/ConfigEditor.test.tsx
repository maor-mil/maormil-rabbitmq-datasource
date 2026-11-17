import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { DataSourceSettings } from '@grafana/data';

import { ConfigEditor } from './ConfigEditor';
import { DEFAULT_OPTIONS, withDefaults } from './defaults';
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

const textbox = (name: string) => screen.getByRole('textbox', { name });

describe('ConfigEditor', () => {
  it('shows the defaults for a new data source without writing on mount', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={settings({})} onOptionsChange={onOptionsChange} />);

    expect(textbox('Host')).toHaveValue(DEFAULT_OPTIONS.host);
    expect(textbox('AMQP Port')).toHaveValue(String(DEFAULT_OPTIONS.amqpPort));
    expect(textbox('Stream Name')).toHaveValue(DEFAULT_OPTIONS.streamOptions.streamName);
    // Grafana 13 drops updates made while the settings page initialises, so the editor must not rely on them.
    expect(onOptionsChange).not.toHaveBeenCalled();
  });

  it('writes the complete settings (defaults included) on the first change of a new data source', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={settings({})} onOptionsChange={onOptionsChange} />);

    fireEvent.change(textbox('Host'), { target: { value: 'broker' } });

    expect(onOptionsChange).toHaveBeenCalledTimes(1);
    const updated: Settings = onOptionsChange.mock.calls[0][0];
    expect(updated.jsonData).toEqual({ ...DEFAULT_OPTIONS, host: 'broker' });
  });

  it('keeps all other settings when a nested stream setting changes', () => {
    const onOptionsChange = jest.fn();
    render(
      <ConfigEditor
        options={settings({ ...DEFAULT_OPTIONS, host: 'broker' }, { password: true })}
        onOptionsChange={onOptionsChange}
      />
    );

    fireEvent.change(textbox('Stream Name'), { target: { value: 'my.stream' } });

    const updated: Settings = onOptionsChange.mock.calls[0][0];
    expect(updated.jsonData).toEqual({
      ...DEFAULT_OPTIONS,
      host: 'broker',
      streamOptions: { ...DEFAULT_OPTIONS.streamOptions, streamName: 'my.stream' },
    });
  });
});

describe('withDefaults', () => {
  it('fills missing top-level and nested stream values without overriding existing ones', () => {
    const filled = withDefaults({ host: 'broker', streamOptions: { streamName: 'custom', maxAge: 0 } as any });
    expect(filled.host).toBe('broker');
    expect(filled.amqpPort).toBe(DEFAULT_OPTIONS.amqpPort);
    expect(filled.streamOptions.streamName).toBe('custom');
    expect(filled.streamOptions.maxAge).toBe(0);
    expect(filled.streamOptions.maxLengthBytes).toBe(DEFAULT_OPTIONS.streamOptions.maxLengthBytes);
  });

  it('returns the defaults for empty settings', () => {
    expect(withDefaults({})).toEqual(DEFAULT_OPTIONS);
    expect(withDefaults(undefined)).toEqual(DEFAULT_OPTIONS);
  });
});
