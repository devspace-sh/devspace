import React from 'react';
import { storiesOf } from '@storybook/react';
import { PortletSimple } from './PortletSimple';

storiesOf('components/advanced/PortletSimple', module).add('default', () => (
  <PortletSimple>
    {{
      top: {
        left: <div>Test</div>,
        right: <div>Test</div>,
      },
    }}
  </PortletSimple>
));
