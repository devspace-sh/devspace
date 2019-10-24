import React from 'react';
import { DevSpaceConfig, DevSpaceConfigContextProvider } from './withDevSpaceConfig';
import ErrorMessage from 'components/basic/ErrorMessage/ErrorMessage';
import { ApiHostname } from 'lib/rest';
import Button from 'components/basic/Button/Button';
import style from './DevSpaceConfigWrapper.module.scss';

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

      const devSpaceConfig: DevSpaceConfig = await response.json();
      devSpaceConfig.changeNamespace = this.changeNamespace;

      this.setState({
        error: null,
        devSpaceConfig,
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

  changeNamespace = (newNamespace: string) => {
    this.setState({
      devSpaceConfig: {
        ...this.state.devSpaceConfig,
        kubeNamespace: newNamespace,
      },
    });
  };

  render() {
    if (this.state.error) {
      return (
        <div className={style['error']}>
          <ErrorMessage className={style['message']}>{this.state.error}</ErrorMessage>
          <div>
            <Button onClick={() => this.componentDidMount()}>Retry</Button>
          </div>
        </div>
      );
    } else if (!this.state.devSpaceConfig) {
      return null;
    }

    return (
      <DevSpaceConfigContextProvider value={this.state.devSpaceConfig}>{this.props.children}</DevSpaceConfigContextProvider>
    );
  }
}
