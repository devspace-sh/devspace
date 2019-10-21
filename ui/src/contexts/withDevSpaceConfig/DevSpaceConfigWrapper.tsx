import React from 'react';
import { DevSpaceConfig, DevSpaceConfigContextProvider } from './withDevSpaceConfig';
import ErrorMessage from 'components/basic/ErrorMessage/ErrorMessage';
import { ApiHostname } from 'lib/rest';

interface Props {}

interface State {
  error: Error;
  devSpaceConfig: DevSpaceConfig;
}

export default class DevSpaceConfigWrapper extends React.PureComponent<Props, State> {
  state: State = {
    error: null,
    devSpaceConfig: null,
  };

  async componentDidMount() {
    try {
      const response = await fetch(`http://${ApiHostname()}/api/config`);
      if (response.status !== 200) {
        this.setState({
          error: new Error(await response.text()),
        });
        return;
      }

      this.setState({
        devSpaceConfig: await response.json(),
      });
    } catch (err) {
      if (err && err.message === 'Failed to fetch') {
        err = new Error('Failed to fetch DevSpace config. Is the UI server running?');
      }

      this.setState({
        error: err,
      });
    }
  }

  render() {
    if (this.state.error) {
      return <ErrorMessage>{this.state.error}</ErrorMessage>;
    } else if (!this.state.devSpaceConfig) {
      return null;
    }

    return (
      <DevSpaceConfigContextProvider value={this.state.devSpaceConfig}>{this.props.children}</DevSpaceConfigContextProvider>
    );
  }
}
