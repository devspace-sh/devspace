import React from 'react';
import { storiesOf } from '@storybook/react';
import CodeSnippet from './CodeSnippet';
import SimpleCodeLine from './SimpleCodeLine/SimpleCodeLine';
import AdvancedCodeLine from './AdvancedCodeLine/AdvancedCodeLine';

storiesOf('components/basic/CodeSnippet', module).add('default', () => (
  <CodeSnippet copy={true}>
    <SimpleCodeLine>
      Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
      aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
      Duis aute irure dolor in reprehenderit.
    </SimpleCodeLine>
    <AdvancedCodeLine>
      At vero eos et accusamus et iusto odio dignissimos ducimus qui blanditiis praesentium voluptatum deleniti atque
      corrupti quos.
    </AdvancedCodeLine>
  </CodeSnippet>
));
