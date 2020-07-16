import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './containers.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import LogsList, { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { V1PodList, V1ServiceList, V1NamespaceList, V1Namespace } from '@kubernetes/client-node';
import Loading from 'components/basic/Loading/Loading';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import LogsLinkTabSelector from 'components/basic/LinkTabSelector/LogsLinkTabSelector/LogsLinkTabSelector';
import TerminalCache from 'components/views/Logs/TerminalCache/TerminalCache';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';
import ChangeNamespace from 'components/views/Logs/ChangeNamespace/ChangeKubeContext';
import authFetch from "../../lib/fetch";

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  serviceList?: V1ServiceList;
  selected?: SelectedLogs;
  namespaceList: V1Namespace[];
}

class LogsContainers extends React.PureComponent<Props, State> {
  timeout: any;
  state: State = {
    namespaceList: null,
  };

  fetchPods = async () => {
    const response = await authFetch(
      `/api/resource?resource=pods&context=${this.props.devSpaceConfig.kubeContext}&namespace=${
        this.props.devSpaceConfig.kubeNamespace
      }`
    );
    if (response.status !== 200) {
      throw new Error(await response.text());
    }

    const podList = await response.json();
    if (!this.state.podList || JSON.stringify(this.state.podList.items) !== JSON.stringify(podList.items)) {
      this.setState({
        podList,
      });
    }
  };

  fetchServices = async () => {
    const response = await authFetch(
      `/api/resource?resource=services&context=${this.props.devSpaceConfig.kubeContext}&namespace=${
        this.props.devSpaceConfig.kubeNamespace
      }`
    );
    if (response.status !== 200) {
      throw new Error(await response.text());
    }

    const serviceList = await response.json();
    if (!this.state.serviceList || JSON.stringify(this.state.serviceList.items) !== JSON.stringify(serviceList.items)) {
      this.setState({
        serviceList,
      });
    }
  };

  fetchNamespaces = async () => {
    try {
      const response = await authFetch(
        `/api/resource?resource=namespaces&context=${this.props.devSpaceConfig.kubeContext}`
      );
      if (response.status !== 200) {
        throw new Error(await response.text());
      }

      const namespaces: V1NamespaceList = await response.json();
      this.setState({ namespaceList: namespaces.items });
    } catch (err) {
      this.setState({ namespaceList: null });
    }
  };

  componentDidMount = async () => {
    try {
      await this.fetchPods();
      await this.fetchServices();
      await this.fetchNamespaces();

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

    this.timeout = setTimeout(this.componentDidMount, 1500);
  };

  componentDidUpdate(prevProps: Props) {
    if (
      prevProps &&
      (this.props.devSpaceConfig.kubeNamespace !== prevProps.devSpaceConfig.kubeNamespace ||
        this.props.devSpaceConfig.kubeContext !== prevProps.devSpaceConfig.kubeContext)
    ) {
      this.setState({
        selected: null,
      });
    }
  }

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  render() {
    return (
      <PageLayout className={styles['logs-containers-component']} heading={<LogsLinkTabSelector />}>
        <TerminalCache
          selected={this.state.selected}
          podList={this.state.podList}
          onDelete={(selected: SelectedLogs) => {
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
          }}
        >
          {({ terminals, cache }) => (
            <React.Fragment>
              {terminals}
              {!this.state.selected && (
                <div className={styles['nothing-selected']}>
                  Please select a container on the right side to display a terminal
                </div>
              )}
              <div className={styles['info-part']}>
                <ChangeNamespace namespaceList={this.state.namespaceList} refetchNamespaces={() => this.fetchNamespaces()} />
                {this.state.podList ? (
                  <LogsList
                    cache={cache}
                    serviceList={this.state.serviceList}
                    podList={this.state.podList}
                    onSelect={(selected: SelectedLogs) => {
                      if (JSON.stringify(selected) === JSON.stringify(this.state.selected)) {
                        selected = null;
                      }

                      this.setState({ selected });
                    }}
                    selected={this.state.selected}
                  />
                ) : (
                  <Loading />
                )}
              </div>
            </React.Fragment>
          )}
        </TerminalCache>
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(LogsContainers))));
