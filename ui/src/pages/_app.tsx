import * as React from 'react';
import Routes from 'routes/Routes';
import PopupWrapper from 'contexts/withPopup/PopupWrapper';
import DevSpaceConfigWrapper from 'contexts/withDevSpaceConfig/DevSpaceConfigWrapper';

interface Props {}
interface State {
  redirect: boolean;
}

export default class App extends React.PureComponent<Props, State> {
  state: State = {
    redirect: false,
  };

  render() {
    if (this.state.redirect) {
      return null;
    }

    return (
      <DevSpaceConfigWrapper>
        <PopupWrapper>
          <Routes />
        </PopupWrapper>
      </DevSpaceConfigWrapper>
    );
  }
}
