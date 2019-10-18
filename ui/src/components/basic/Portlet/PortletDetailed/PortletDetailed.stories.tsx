import React from 'react';
import { storiesOf } from '@storybook/react';
import { PortletDetailed } from './PortletDetailed';

storiesOf('components/advanced/PortletDetailed', module).add('default', () => (
  <PortletDetailed>
    {{
      top: {
        left: <div>Test</div>,
        right: <div>Test</div>,
      },
      bottom: {
        left: <div>Test</div>,
      },
    }}
  </PortletDetailed>
));
