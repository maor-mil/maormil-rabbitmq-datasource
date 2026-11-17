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

describe('ConfigEditor TLS', () => {
  const configured = (patch: Partial<RabbitMQDataSourceOptions>, secure = {}) =>
    settings({ ...DEFAULT_OPTIONS, ...patch }, { password: true, ...secure });

  it('only shows the TLS settings when "TLS Connection" is on', () => {
    const { rerender } = render(<ConfigEditor options={configured({})} onOptionsChange={jest.fn()} />);
    expect(screen.queryByText('TLS Settings')).not.toBeInTheDocument();

    rerender(<ConfigEditor options={configured({ tlsConnection: true })} onOptionsChange={jest.fn()} />);
    expect(screen.getByText('TLS Settings')).toBeInTheDocument();
    expect(screen.queryByLabelText('CA Cert')).not.toBeInTheDocument();

    rerender(
      <ConfigEditor options={configured({ tlsConnection: true, tlsAuthWithCACert: true, tlsAuth: true })} onOptionsChange={jest.fn()} />
    );
    expect(screen.getByLabelText('CA Cert')).toBeInTheDocument();
    expect(screen.getByLabelText('Client Cert')).toBeInTheDocument();
    expect(screen.getByLabelText('Client Key')).toBeInTheDocument();
  });

  it('switches the standard ports when TLS is toggled, but keeps custom ports', () => {
    const onOptionsChange = jest.fn();
    const { rerender } = render(<ConfigEditor options={configured({})} onOptionsChange={onOptionsChange} />);
    fireEvent.click(screen.getByRole('switch', { name: 'TLS Connection' }));
    expect(onOptionsChange.mock.calls[0][0].jsonData).toMatchObject({ tlsConnection: true, amqpPort: 5671, streamPort: 5551 });

    onOptionsChange.mockClear();
    rerender(<ConfigEditor options={configured({ amqpPort: 1234, streamPort: 5552 })} onOptionsChange={onOptionsChange} />);
    fireEvent.click(screen.getByRole('switch', { name: 'TLS Connection' }));
    expect(onOptionsChange.mock.calls[0][0].jsonData).toMatchObject({ amqpPort: 1234, streamPort: 5551 });
  });

  it('keeps unsaved secrets when another secret changes', () => {
    const onOptionsChange = jest.fn();
    const options = configured({ tlsConnection: true, tlsAuthWithCACert: true });
    options.secureJsonData = { tlsCACert: 'CA PEM' };
    render(<ConfigEditor options={options} onOptionsChange={onOptionsChange} />);

    fireEvent.change(screen.getByLabelText('Password', { selector: 'input' }), { target: { value: 'new-password' } });

    expect(onOptionsChange.mock.calls[0][0].secureJsonData).toEqual({ tlsCACert: 'CA PEM', password: 'new-password' });
  });

  it('hides username and password for EXTERNAL authentication', () => {
    render(<ConfigEditor options={configured({ tlsConnection: true, saslMechanism: 'EXTERNAL' })} onOptionsChange={jest.fn()} />);
    expect(screen.queryByRole('textbox', { name: 'Username' })).not.toBeInTheDocument();
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

  it('drops the dispose settings that older versions stored', () => {
    const filled = withDefaults({
      streamOptions: { streamName: 's', shouldDisposeStream: true } as any,
      exchangesOptions: [{ name: 'e', shouldDisposeExchange: true, disposeIfUnused: true } as any],
      bindingsOptions: [{ senderName: 'e', shouldDisposeBinding: true } as any],
    });
    expect(filled.streamOptions).not.toHaveProperty('shouldDisposeStream');
    expect(filled.exchangesOptions[0]).toEqual({ name: 'e' });
    expect(filled.bindingsOptions[0]).toEqual({ senderName: 'e' });
  });

  it('returns the defaults for empty settings', () => {
    expect(withDefaults({})).toEqual(DEFAULT_OPTIONS);
    expect(withDefaults(undefined)).toEqual(DEFAULT_OPTIONS);
  });
});
