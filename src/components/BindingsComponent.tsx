import React from 'react';

import { InlineField, InlineSwitch, Input, Button } from '@grafana/ui';

import { BindingsOptions, BindingOptions } from '../types';
import { LABEL_WIDTH, INPUT_WIDTH, SWITCH_WIDTH } from './consts';


export function BindingsComponent({ bindings, setBindings }: { bindings: BindingsOptions, setBindings: React.Dispatch<React.SetStateAction<BindingsOptions>>}) {
    const DEFAULT_SHOULD_DISPOSE_BINDING = false;
    const DEFAULT_IS_QUEUE_BINDING = true;
    const DEFAULT_SENDER_NAME = "rabbitmq.exchange";
    const DEFAULT_ROUTING_KEY = "/";
    const DEFAULT_RECEIVER_NAME = "rabbitmq.stream";
    const DEFAULT_NO_WAIT = false;

    const addBinding = () => {
        const newBinding: BindingOptions = {
            shouldDisposeBinding: DEFAULT_SHOULD_DISPOSE_BINDING,
            isQueueBinding: DEFAULT_IS_QUEUE_BINDING,
            senderName: DEFAULT_SENDER_NAME,
            routingKey: DEFAULT_ROUTING_KEY,
            receiverName: DEFAULT_RECEIVER_NAME,
            noWait: DEFAULT_NO_WAIT
        }

        setBindings(prevBindings => [...prevBindings, newBinding]);
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
                <InlineField label="Should Dispose Binding" labelWidth={LABEL_WIDTH} tooltip="Should unbind when the RabbitMQ datasource is deleted">
                    <InlineSwitch
                        onChange={ event => updatebindingProperty(index, 'shouldDisposeBinding', event.currentTarget.checked)}
                        value={value.shouldDisposeBinding}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Is Queue Binding" labelWidth={LABEL_WIDTH} tooltip="Should binding be from Exchange to queue/stream (if disabled, the binding will be from exchange to exchange">
                    <InlineSwitch
                        onChange={ event => updatebindingProperty(index, 'isQueueBinding', event.currentTarget.checked)}
                        value={value.isQueueBinding}
                        width={SWITCH_WIDTH}
                    />
                </InlineField>
                <InlineField label="Sender Name" labelWidth={LABEL_WIDTH} tooltip="The exchange to bind from">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'senderName', event.currentTarget.value || DEFAULT_SENDER_NAME)}
                        value={value.senderName}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Routing Key" labelWidth={LABEL_WIDTH} tooltip="The routing key to bind between the sender exchange and the receiver">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'routingKey', event.currentTarget.value || DEFAULT_ROUTING_KEY)}
                        value={value.routingKey}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Receiver Name" labelWidth={LABEL_WIDTH} tooltip="The stream/queue/exchange to bind to">
                    <Input
                        onChange={ event => updatebindingProperty(index, 'receiverName', event.currentTarget.value || DEFAULT_RECEIVER_NAME)}
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
        <Button variant="secondary" fill="text" icon="plus" onClick={addBinding} tooltip="Add" aria-label="Add" />
    </>)
};
