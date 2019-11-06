import * as React from 'react';
import Routes from 'routes/Routes';
import PopupWrapper from 'contexts/withPopup/PopupWrapper';
import DevSpaceConfigWrapper from 'contexts/withDevSpaceConfig/DevSpaceConfigWrapper';
import WarningWrapper from 'contexts/withWarning/WarningWrapper';
import SupportChat from 'components/advanced/SupportChat/SupportChat';

interface Props {}
interface State {}

export default class App extends React.PureComponent<Props, State> {
  state: State = {};

  render() {
    return (
      <DevSpaceConfigWrapper>
        <PopupWrapper>
          <WarningWrapper>
            <Routes />
            <SupportChat />
          </WarningWrapper>
        </PopupWrapper>
      </DevSpaceConfigWrapper>
    );
  }
}
