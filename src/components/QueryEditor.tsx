import React from 'react';

import { InlineLabel } from '@grafana/ui';
import { SHORT_TEXT_WIDTH } from './consts';

export const QueryEditor = () => {
  return (
    <>
      <InlineLabel width={SHORT_TEXT_WIDTH} tooltip="The Consumner of the RabbitMQ Stream is Created Once You Press Reset Query!">
      Ready to go!
      </InlineLabel>
    </>
  );
}
