import React from 'react';
import { DevSpaceConfig, DevSpaceConfigContextProvider } from './withDevSpaceConfig';
import ErrorMessage from 'components/basic/ErrorMessage/ErrorMessage';
import Button from 'components/basic/Button/Button';
import styles from './DevSpaceConfigWrapper.module.scss';
import authFetch from "../../lib/fetch";

interface Props {}

interface State {
  error: Error;
  devSpaceConfig: DevSpaceConfig;
}

export interface NewContext {
  contextName?: string;
  contextNamespace?: string;
}

export default class DevSpaceConfigWrapper extends React.PureComponent<Props, State> {
  state: State = {
    error: null,
    devSpaceConfig: null,
  };

  async componentDidMount() {
    try {
      const response = await authFetch(`/api/config`);
      if (response.status !== 200) {
        this.setState({
          error: new Error(await response.text()),
        });
        return;
      }

      const devSpaceConfig: DevSpaceConfig = await response.json();

      devSpaceConfig.changeKubeContext = this.changeKubeContext;
      devSpaceConfig.originalKubeContext = devSpaceConfig.kubeContext;
      devSpaceConfig.originalKubeNamespace = devSpaceConfig.kubeNamespace;

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

  changeKubeContext = (context: NewContext) => {
    if (context.contextName) {
      this.setState({
        devSpaceConfig: {
          ...this.state.devSpaceConfig,
          kubeNamespace: context.contextNamespace,
          kubeContext: context.contextName,
        },
      });
    } else {
      this.setState({
        devSpaceConfig: {
          ...this.state.devSpaceConfig,
          kubeNamespace: context.contextNamespace,
        },
      });
    }
  };

  render() {
    if (this.state.error) {
      return (
        <div className={styles['error']}>
          <ErrorMessage className={styles['message']}>{this.state.error}</ErrorMessage>
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
