import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from 'pages/styles/logs.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import LogsList, { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { V1PodList } from '@kubernetes/client-node';
import Loading from 'components/basic/Loading/Loading';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import ErrorMessage from 'components/basic/ErrorMessage/ErrorMessage';
import { ApiHostname } from 'lib/rest';
import LogsLinkTabSelector from 'components/basic/LinkTabSelector/LogsLinkTabSelector/LogsLinkTabSelector';
import TerminalCache from 'lib/TerminalCache';

interface Props extends DevSpaceConfigContext, PopupContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: SelectedLogs;
  error?: Error;
}

class LogsContainers extends React.PureComponent<Props, State> {
  timeout: any;
  cache: TerminalCache = new TerminalCache(this.props.devSpaceConfig.kubeNamespace, (selected: SelectedLogs) => {
    if (
      this.state.selected &&
      selected.pod === this.state.selected.pod &&
      selected.container === this.state.selected.container
    ) {
      this.setState({
        selected: null,
      });
      return;
    } else if (this.state.selected && selected.multiple && this.state.selected.multiple) {
      this.setState({
        selected: null,
      });
      return;
    }

    this.forceUpdate();
  });
  state: State = {};

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

      const podList = await response.json();

      this.cache.updateCache(podList);
      this.setState({
        error: null,
        podList,
      });

      // this.timeout = setTimeout(this.componentDidMount, 1000);
    } catch (err) {
      if (err && err.message === 'Failed to fetch') {
        err = new Error('Failed to fetch pods. Is the UI server running?');
      }

      this.setState({
        error: err,
      });
    }
  };

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  renderTerminal() {
    return <div>{this.cache.renderTerminals()}</div>;
  }

  render() {
    return (
      <PageLayout className={styles['spaces-component']} heading={<LogsLinkTabSelector />}>
        {this.state.error ? (
          <ErrorMessage>{this.state.error}</ErrorMessage>
        ) : this.state.podList ? (
          <LogsList
            podList={this.state.podList}
            onSelect={(selected: SelectedLogs) => {
              if (JSON.stringify(selected) === JSON.stringify(this.state.selected)) {
                selected = null;
              }

              this.cache.select(selected);
              this.setState({ selected });
            }}
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
