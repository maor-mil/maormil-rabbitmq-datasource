import React from 'react';

import { InlineLabel } from '@grafana/ui';
import { SHORT_TEXT_WIDTH } from './consts';

export const QueryEditor = () => {
  return (
    <>
      <InlineLabel width={SHORT_TEXT_WIDTH} tooltip="The Consumer of the RabbitMQ Stream is Created Once You Press Refresh Dashboard!">
      Ready to go!
      </InlineLabel>
    </>
  );
}
