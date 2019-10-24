import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './configuration.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { V1PodList } from '@kubernetes/client-node';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';
import StackLinkTabSelector from 'components/basic/LinkTabSelector/StackLinkTabSelector/StackLinkTabSelector';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: SelectedLogs;
}

class StackConfiguration extends React.PureComponent<Props, State> {
  state: State = {};

  renderTerminal = () => {
    return <CodeSnippet>d</CodeSnippet>;
  };

  render() {
    return (
      <PageLayout className={styles['stack-configuration-component']} heading={<StackLinkTabSelector />}>
        {this.renderTerminal()}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(StackConfiguration))));
