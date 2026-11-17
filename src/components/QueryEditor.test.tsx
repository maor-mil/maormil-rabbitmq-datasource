import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';

import { QueryEditor } from './QueryEditor';
import { RabbitMQQuery } from '../types';

const renderEditor = (query: Partial<RabbitMQQuery>) => {
  const onChange = jest.fn();
  const onRunQuery = jest.fn();
  const props = { query: { refId: 'A', ...query }, onChange, onRunQuery } as any;
  render(<QueryEditor {...props} />);
  return { onChange, onRunQuery };
};

describe('QueryEditor', () => {
  it('defaults to the data source stream without extra fields', () => {
    renderEditor({});
    expect(screen.getByRole('radio', { name: 'Data source stream' })).toBeChecked();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
  });

  it('switching the source updates the query and runs it', () => {
    const { onChange, onRunQuery } = renderEditor({});
    fireEvent.click(screen.getByRole('radio', { name: 'Queue' }));
    expect(onChange).toHaveBeenCalledWith({ refId: 'A', source: 'queue' });
    expect(onRunQuery).toHaveBeenCalled();
  });

  it('shows the stream name and offset for a stream source', () => {
    renderEditor({ source: 'stream', stream: 'orders' });
    expect(screen.getByRole('textbox', { name: 'Stream' })).toHaveValue('orders');
    expect(screen.getByRole('radio', { name: 'First' })).toBeChecked();
  });

  it('runs text field changes on blur only', () => {
    const { onChange, onRunQuery } = renderEditor({ source: 'stream' });
    const input = screen.getByRole('textbox', { name: 'Stream' });
    fireEvent.change(input, { target: { value: 'orders' } });
    expect(onChange).toHaveBeenCalledWith({ refId: 'A', source: 'stream', stream: 'orders' });
    expect(onRunQuery).not.toHaveBeenCalled();
    fireEvent.blur(input);
    expect(onRunQuery).toHaveBeenCalled();
  });

  it('taps an exchange by default and warns before consuming an existing queue', () => {
    renderEditor({ source: 'queue' });
    expect(screen.getByRole('textbox', { name: 'Exchange' })).toBeInTheDocument();
    expect(screen.queryByText('Consuming a queue removes its messages')).not.toBeInTheDocument();

    renderEditor({ source: 'queue', queueMode: 'existing' });
    expect(screen.getByText('Consuming a queue removes its messages')).toBeInTheDocument();
  });

  it('edits Bloom filter values, match unfiltered and the SQL filter of a stream source', () => {
    const { onChange, onRunQuery } = renderEditor({ source: 'stream', stream: 's', filterValues: ['emea'] });
    expect(screen.getByText('emea')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('switch', { name: 'Match unfiltered' }));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ matchUnfiltered: true }));

    const sql = screen.getByRole('textbox', { name: 'SQL filter' });
    fireEvent.change(sql, { target: { value: "region = 'EMEA'" } });
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ sql: "region = 'EMEA'" }));
    fireEvent.blur(sql);
    expect(onRunQuery).toHaveBeenCalled();
  });

  it('only offers match unfiltered once there are filter values', () => {
    renderEditor({ source: 'stream', stream: 's' });
    expect(screen.queryByRole('switch', { name: 'Match unfiltered' })).not.toBeInTheDocument();
  });
});
