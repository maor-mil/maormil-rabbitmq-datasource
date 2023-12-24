import React from 'react';

import { Input, Button, InlineField } from '@grafana/ui';

import { JsonQueryModel, JsonQueryModels } from '../types';
import { LABEL_WIDTH, INPUT_WIDTH } from './consts';


export function JsonQueryModelComponent({ jsonQueryModels, setJsonQueryModels }: { jsonQueryModels: JsonQueryModels, setJsonQueryModels: React.Dispatch<React.SetStateAction<JsonQueryModels>>}) {
    const addJsonQueryModel = () => {
        const newJsonQueryModel: JsonQueryModel = {
            jsonKeyPath: "Your Json Key",
            regexValue: "Your wanted json value with Regex"
        }
    
        setJsonQueryModels(prevJsonQueryModels => [...prevJsonQueryModels, newJsonQueryModel]);
      }
    
    const updateJsonQueryModelProperty = (index: number, property: keyof JsonQueryModel, value: string | boolean) => {
        setJsonQueryModels(prevJsonQueryModels =>
          prevJsonQueryModels.map((JsonQueryModel, i) => (i === index ? { ...JsonQueryModel, [property]: value } : JsonQueryModel))
        );
      };
    
    const removeJsonQueryModel = (index: number) => {
        setJsonQueryModels(prevJsonQueryModels => prevJsonQueryModels.filter((_, i) => i !== index));
    }

    return (
    <>
        {
        jsonQueryModels.map((value, index) => (
            <>
                <InlineField label="Key Name" labelWidth={LABEL_WIDTH} tooltip="The name of the key in the expected JSON">
                    <Input
                        onChange={ event => updateJsonQueryModelProperty(index, 'jsonKeyPath', event.currentTarget.value)}
                        value={value.jsonKeyPath}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <InlineField label="Expected Value" labelWidth={LABEL_WIDTH} tooltip="The expected value using Regex">
                    <Input
                        onChange={ event => updateJsonQueryModelProperty(index, 'regexValue', event.currentTarget.value)}
                        value={value.regexValue}
                        width={INPUT_WIDTH}
                    />
                </InlineField>
                <Button variant="secondary" fill="text" icon="minus" onClick={() => removeJsonQueryModel(index)} tooltip="Remove" aria-label="Remove" />
            </>
            ))
        }
        <Button variant="secondary" fill="text" icon="plus" onClick={addJsonQueryModel} tooltip="Add" aria-label="Add" />
    </>)
}
