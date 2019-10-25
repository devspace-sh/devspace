import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './commands.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { V1PodList } from '@kubernetes/client-node';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';
import CommandsLinkTabSelector from 'components/basic/LinkTabSelector/CommandsLinkTabSelector/CommandsLinkTabSelector';
import CommandsList from 'components/views/Commands/Commands/CommandsList/CommandsList';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: SelectedLogs;
}

class Commands extends React.PureComponent<Props, State> {
  state: State = {};

  renderConfig = () => {
    return (
      <CodeSnippet lineNumbers={true} className={styles.codesnippet}>
        blabla
      </CodeSnippet>
    );
  };

  render() {
    console.log(this.props.devSpaceConfig);
    return (
      <PageLayout className={styles['commands-component']} heading={<CommandsLinkTabSelector />}>
        {!this.props.devSpaceConfig.rawConfig ||
        !this.props.devSpaceConfig.rawConfig.commands ||
        this.props.devSpaceConfig.rawConfig.commands.length === 0 ? (
          <div className={styles['no-config']}>There is no command available. </div>
        ) : (
          <React.Fragment>
            {this.renderConfig()}
            <div className={styles['info-part']}>
              <CommandsList commandsList={this.props.devSpaceConfig.rawConfig.commands} />
            </div>
          </React.Fragment>
        )}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(Commands))));
