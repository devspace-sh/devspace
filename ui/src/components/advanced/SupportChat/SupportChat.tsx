import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';

interface Props extends DevSpaceConfigContext {}
interface State {}

class SupportChat extends React.PureComponent<Props, State> {

    render() {
      let optionalComponent;
  
      try {
        const Chat = require("@devspace/react-components").Chat;
        const Analytics = require("@devspace/react-components").Analytics;
  
        optionalComponent = (
          <div>
            <Chat />
            {this.props.devSpaceConfig.analyticsEnabled &&
              <Analytics />
            }
          </div>
        );
      } catch (e) {
        console.log("Not loading optional components in dev mode.")
      }
  
      return (
        <div>
  
          {optionalComponent}
  
          <noscript>
            <iframe
              src="https://www.googletagmanager.com/ns.html?id=GTM-KM6KSWG"
              height="0"
              width="0"
              style={{ display: "none", visibility: "hidden" }}
            />
          </noscript>
        </div>
      );
    }
}

export default withDevSpaceConfig(SupportChat);
