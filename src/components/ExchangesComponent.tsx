import React from 'react';

import { InlineField, InlineSwitch, Input, Button } from '@grafana/ui';

import { Exchanges, ExchangeOptions } from '../types';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';


export function ExchangesComponent({ exchanges, setExchanges }: { exchanges: Exchanges, setExchanges: React.Dispatch<React.SetStateAction<Exchanges>>}) {
    const addExchange = () => {
        const newExchange: ExchangeOptions = {
            name: "Type the Exchange name",
            type: "fanout",
            durable: true,
            autoDeleted: false,
            internal: false,
            noWait: false
        }

        setExchanges(prevExchanges => [...prevExchanges, newExchange]);
    }

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
                <InlineField label="Exchange Name" labelWidth={LABEL_WIDTH} tooltip="Exchange that need to be created in the RabbitMQ">
                    <Input
                        onChange={ event => updateExchangeProperty(index, 'name', event.currentTarget.value)}
                        value={value.name}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Exchange Type" labelWidth={LABEL_WIDTH} tooltip="Exchange Type (should only accept: fanout | direct | topic | headers | x-consistent-hash)">
                    <Input
                        onChange={ event => updateExchangeProperty(index, 'type', event.currentTarget.value)}
                        value={value.type}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Durable" labelWidth={LABEL_WIDTH} tooltip="Should exchange be durable">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'durable', event.currentTarget.checked)}
                        value={value.durable}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Auto Deleted" labelWidth={LABEL_WIDTH} tooltip="Should exchange be auto deleted">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'autoDeleted', event.currentTarget.checked)}
                        value={value.autoDeleted}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Internal" labelWidth={LABEL_WIDTH} tooltip="Should exchange be internal">
                    <InlineSwitch
                        onChange={ event => updateExchangeProperty(index, 'internal', event.currentTarget.checked)}
                        value={value.internal}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is No Wait" labelWidth={LABEL_WIDTH} tooltip="Should exchange be noWait">
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
