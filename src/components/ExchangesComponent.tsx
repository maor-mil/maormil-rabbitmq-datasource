import React from 'react';

import { InlineField, InlineSwitch, Input, Button, RadioButtonGroup } from '@grafana/ui';

import { ExchangesOptions, ExchangeOptions } from '../types';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';


export function ExchangesComponent({ exchanges, setExchanges }: { exchanges: ExchangesOptions, setExchanges: React.Dispatch<React.SetStateAction<ExchangesOptions>>}) {
    const DEFAULT_SHOULD_DISPOSE_EXCHANGE = true;
    const DEFAULT_DISPOSE_IF_UNUSED = true;
    const DEFAULT_NAME = "rabbitmq.exchange";
    const DEFAULT_TYPE = "fanout";
    const DEFAULT_DURABLE = true;
    const DEFAULT_AUTO_DELETED = false;
    const DEFAULT_INTERNAL = false;
    const DEFAULT_NO_WAIT = false;
    

    const addExchange = () => {
        const newExchange: ExchangeOptions = {
            shouldDisposeExchange: DEFAULT_SHOULD_DISPOSE_EXCHANGE,
            disposeIfUnused: DEFAULT_DISPOSE_IF_UNUSED,
            name: DEFAULT_NAME,
            type: DEFAULT_TYPE,
            durable: DEFAULT_DURABLE,
            autoDeleted: DEFAULT_AUTO_DELETED,
            internal: DEFAULT_INTERNAL,
            noWait: DEFAULT_NO_WAIT
        }

        setExchanges(prevExchanges => [...prevExchanges, newExchange]);
    }

    const exchangeTypes = [{
        label: 'Fanout',
        value: 'fanout'
      }, {
        label: 'Direct',
        value: 'direct'
      }, {
        label: 'Topic',
        value: 'topic'
      }, {
        label: 'Headers',
        value: 'headers'
      }, {
        label: 'X Consistent Hash',
        value: 'x-consistent-hash'
      },
    ];

    const updateExchangeProperty = (index: number, property: keyof ExchangeOptions, value: string | boolean) => {
        setExchanges(prevExchanges =>
          prevExchanges.map((exchange, i) => (i === index ? { ...exchange, [property]: value } : exchange))
        );
      };

    const removeExchange = (index: number) => {
        setExchanges(prevExchanges => prevExchanges.filter((_, i) => i !== index));
    }
    
    return (
    <>
        {
        exchanges.map((value, index) => (
            <>
                <InlineField label="Should Dispose Exchange" labelWidth={LABEL_WIDTH} tooltip="Should delete this exchange when the RabbitMQ datasource is deleted">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'shouldDisposeExchange', event.currentTarget.checked)}
                        value={value.shouldDisposeExchange}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Dispose if Unused" labelWidth={LABEL_WIDTH} tooltip="Delete this exchange only if it doesn't have bindings (and if 'Should Dispose Exchange' is set ON)">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'disposeIfUnused', event.currentTarget.checked)}
                        value={value.disposeIfUnused}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Exchange Name" labelWidth={LABEL_WIDTH} tooltip="The exchange name that should exist in the RabbitMQ">
                    <Input
                        onChange={ event => updateExchangeProperty(index, 'name', event.currentTarget.value || DEFAULT_NAME)}
                        value={value.name}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Exchange Type" labelWidth={LABEL_WIDTH} tooltip="The exchange type">
                    <RadioButtonGroup options={exchangeTypes} value={value.type} onChange={event => updateExchangeProperty(index, 'type', event)} />
                </InlineField>
                <InlineField label="Is Durable" labelWidth={LABEL_WIDTH} tooltip="Should the exchange be durable">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'durable', event.currentTarget.checked)}
                        value={value.durable}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Auto Deleted" labelWidth={LABEL_WIDTH} tooltip="Should the exchange be auto deleted">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'autoDeleted', event.currentTarget.checked)}
                        value={value.autoDeleted}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Internal" labelWidth={LABEL_WIDTH} tooltip="Should the exchange be internal">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'internal', event.currentTarget.checked)}
                        value={value.internal}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is No Wait" labelWidth={LABEL_WIDTH} tooltip="Should the exchange be noWait">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'noWait', event.currentTarget.checked)}
                        value={value.noWait}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <Button variant="secondary" fill="text" icon="minus" onClick={() => removeExchange(index)} tooltip="Remove" aria-label="Remove" />
            </>
            ))
        }
        <Button variant="secondary" fill="text" icon="plus" onClick={addExchange} tooltip="Add" aria-label="Add" />
    </>)
};
