import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from 'pages/styles/logs.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import LogsList from 'components/views/Logs/LogsList/LogsList';
import { V1PodList } from '@kubernetes/client-node';
import Loading from 'components/basic/Loading/Loading';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import ErrorMessage from 'components/basic/ErrorMessage/ErrorMessage';
import { ApiHostname } from 'lib/rest';
import LogsLinkTabSelector from 'components/basic/LinkTabSelector/LogsLinkTabSelector/LogsLinkTabSelector';
import LogsTerminal, { LogsTerminalProps } from 'components/views/Logs/LogsTerminal/LogsTerminal';

interface Props extends DevSpaceConfigContext, PopupContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: {
    pod: string;
    container: string;
  };
  terminalList: LogsTerminalProps[];
  error?: Error;
}

class LogsContainers extends React.PureComponent<Props, State> {
  timeout: any;
  state: State = {
    terminalList: [],
  };

  componentDidMount = async () => {
    try {
      const response = await fetch(
        `http://${ApiHostname()}/api/resource?resource=pods&namespace=${this.props.devSpaceConfig.kubeNamespace}`
      );
      if (response.status !== 200) {
        this.setState({
          error: new Error(await response.text()),
        });
        return;
      }

      this.setState({
        error: null,
        podList: await response.json(),
      });

      this.timeout = setTimeout(this.componentDidMount, 1000);
    } catch (err) {
      this.setState({
        error: err,
      });
    }
  };

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  componentDidUpdate() {
    let selected = this.state.selected;
    let terminals = this.state.terminalList;

    // Check if selected exists
    if (this.state.selected) {
      const selectedPod =
        this.state.podList && this.state.podList.items.find((pod) => this.state.selected.pod === pod.metadata.name);
      if (!selectedPod) {
        selected = null;
        this.setState({ selected: undefined });
      } else {
        const selectedContainer = selectedPod.spec.containers.find(
          (container) => this.state.selected.container === container.name
        );
        if (!selectedContainer) {
          selected = null;
          this.setState({ selected: undefined });
        }
      }
    }

    // Check if we have to update the terminal list
    if (this.state.terminalList && this.state.terminalList.length > 0) {
      const newList = [...this.state.terminalList];
      let changed = false;

      for (let i = 0; i < newList.length; i++) {
        const selectedPod =
          this.state.podList && this.state.podList.items.find((pod) => newList[i].pod === pod.metadata.name);
        if (!selectedPod) {
          newList.splice(i, 1);
          i--;
          continue;
        }

        const selectedContainer = selectedPod.spec.containers.find((container) => newList[i].container === container.name);
        if (!selectedContainer) {
          newList.splice(i, 1);
          i--;
          continue;
        }

        const show = selected && newList[i].pod === selected.pod && newList[i].container === selected.container;
        if (show !== newList[i].show) {
          newList[i] = { ...newList[i], show };
          changed = true;
        }
      }

      if (changed || newList.length < this.state.terminalList.length) {
        terminals = newList;
        this.setState({
          terminalList: newList,
        });
      }
    }

    if (
      selected &&
      !terminals.find((terminal) => terminal.pod === selected.pod && terminal.container === selected.container)
    ) {
      this.setState({
        terminalList: [
          ...this.state.terminalList,
          {
            pod: selected.pod,
            container: selected.container,
            namespace: this.props.devSpaceConfig.kubeNamespace,
            show: true,
          },
        ],
      });
    }
  }

  renderTerminal() {
    return (
      <div>
        {this.state.terminalList.map((terminal) => (
          <LogsTerminal key={terminal.pod + ':' + terminal.container} {...terminal} />
        ))}
      </div>
    );
  }

  render() {
    return (
      <PageLayout className={styles['spaces-component']} heading={<LogsLinkTabSelector />}>
        {this.state.error ? (
          <ErrorMessage>{this.state.error}</ErrorMessage>
        ) : this.state.podList ? (
          <LogsList
            podList={this.state.podList}
            onSelect={(podName, containerName) => this.setState({ selected: { pod: podName, container: containerName } })}
            selected={this.state.selected}
          />
        ) : (
          <Loading />
        )}
        {this.renderTerminal()}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(LogsContainers)));
