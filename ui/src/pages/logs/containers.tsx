import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './containers.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import LogsList, { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { V1PodList } from '@kubernetes/client-node';
import Loading from 'components/basic/Loading/Loading';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { ApiHostname } from 'lib/rest';
import LogsLinkTabSelector from 'components/basic/LinkTabSelector/LogsLinkTabSelector/LogsLinkTabSelector';
import TerminalCache from 'components/views/Logs/TerminalCache/TerminalCache';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: SelectedLogs;
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
        throw new Error(await response.text());
      }

      const podList = await response.json();
      if (!this.state.podList || JSON.stringify(this.state.podList.items) !== JSON.stringify(podList.items)) {
        this.cache.updateCache(podList);
        this.setState({
          podList,
        });
      }

      if (
        this.props.warning.getActive() &&
        typeof this.props.warning.getActive().children === 'string' &&
        this.props.warning
          .getActive()
          .children.toString()
          .indexOf('Containers:') === 0
      ) {
        this.props.warning.close();
      }
    } catch (err) {
      let message = err.message;
      if (message === 'Failed to fetch') {
        message = 'Containers: Failed to fetch pods. Is the UI server running?';
      } else {
        message = 'Containers: Error retrieving pods: ' + message;
      }

      if (!this.props.warning.getActive()) {
        this.props.warning.show(message);
      }
    }

    this.timeout = setTimeout(this.componentDidMount, 1000);
  };

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  renderTerminal() {
    return (
      <React.Fragment>
        {!this.state.selected && (
          <div className={styles['nothing-selected']}>Please select container on the left side to display a terminal</div>
        )}
        {this.cache.renderTerminals()}
      </React.Fragment>
    );
  }

  render() {
    return (
      <PageLayout className={styles['logs-containers-component']} heading={<LogsLinkTabSelector />}>
        {this.state.podList ? (
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

export default withRouter(withPopup(withDevSpaceConfig(withWarning(LogsContainers))));
