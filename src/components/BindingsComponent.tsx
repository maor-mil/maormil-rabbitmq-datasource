import React from 'react';

import { InlineField, InlineSwitch, Input, Button } from '@grafana/ui';

import { Bindings, BindingOptions } from '../types';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';


export function BindingsComponent({ bindings, setBindings }: { bindings: Bindings, setBindings: React.Dispatch<React.SetStateAction<Bindings>>}) {
    const addbinding = () => {
        const newbinding: BindingOptions = {
            isQueueBinding: true,
            senderName: "Some exchange in your RabbitMQ",
            routingKey: "/",
            receiverName: "Probably your stream name or some another exchange in your RabbitMQ",
            noWait: false
        }

        setBindings(prevBindings => [...prevBindings, newbinding]);
    }

    const updatebindingProperty = (index: number, property: keyof BindingOptions, value: string | boolean) => {
        setBindings(prevbindings =>
          prevbindings.map((binding, i) => (i === index ? { ...binding, [property]: value } : binding))
        );
      };

    const removebinding = (index: number) => {
        setBindings(prevbindings => prevbindings.filter((_, i) => i !== index));
    }
    
    return (
    <>
        {
        bindings.map((value, index) => (
            <>
                <InlineField label="Is Queue Binding" labelWidth={LABEL_WIDTH} tooltip="Should binding be from Exchange to queue/stream (if disabled, the binding will be from exchange to exchange">
                    <InlineSwitch
                        onChange={ event => updatebindingProperty(index, 'isQueueBinding', event.currentTarget.checked)}
                        value={value.isQueueBinding}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Sender Name" labelWidth={LABEL_WIDTH} tooltip="The exchange to bind from">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'senderName', event.currentTarget.value)}
                        value={value.senderName}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Routing Key" labelWidth={LABEL_WIDTH} tooltip="The routing key to bind between the sender exchange and the receiver">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'routingKey', event.currentTarget.value)}
                        value={value.routingKey}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Receiver Name" labelWidth={LABEL_WIDTH} tooltip="The stream/queue/exchange to bind to">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'receiverName', event.currentTarget.value)}
                        value={value.receiverName}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is No Wait" labelWidth={LABEL_WIDTH} tooltip="Should binding be noWait">
                    <InlineSwitch
                        onChange={ event => updatebindingProperty(index, 'noWait', event.currentTarget.checked)}
                        value={value.noWait}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <Button variant="secondary" fill="text" icon="minus" onClick={() => removebinding(index)} tooltip="Remove" aria-label="Remove" />
            </>
            ))
        }
        <Button variant="secondary" fill="text" icon="plus" onClick={addbinding} tooltip="Add" aria-label="Add" />
    </>)
};
