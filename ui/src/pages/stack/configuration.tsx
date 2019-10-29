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
import ProfilePortlet from 'components/views/Stack/Configuration/ProfilePortlet/ProfilePortlet';
import ConfigVariablesPortlet from 'components/views/Stack/Configuration/ConfigVariablesPortlet/ConfigVariablesPortlet';
import { configToYAML } from 'lib/utils';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: SelectedLogs;
}

class StackConfiguration extends React.PureComponent<Props, State> {
  state: State = {};

  renderConfig = () => {
    return (
      <CodeSnippet lineNumbers={true} className={styles.codesnippet}>
        {configToYAML(this.props.devSpaceConfig.config, true)}
      </CodeSnippet>
    );
  };

  render() {
    return (
      <PageLayout className={styles['stack-configuration-component']} heading={<StackLinkTabSelector />}>
        {!this.props.devSpaceConfig.config || !this.props.devSpaceConfig.generatedConfig ? (
          <div className={styles['no-config']}>
            <div>
              There was no DevSpace configuration loaded.&nbsp;
              <a href="https://devspace.cloud/docs/cli/getting-started/deployment" target="_blank">
                Click here
              </a>
              &nbsp;to create a new DevSpace configuration
            </div>
          </div>
        ) : (
          <React.Fragment>
            {this.renderConfig()}
            <div className={styles['info-part']}>
              <ProfilePortlet profile={this.props.devSpaceConfig.profile} />
              <ConfigVariablesPortlet vars={this.props.devSpaceConfig.generatedConfig.vars} />
            </div>
          </React.Fragment>
        )}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(StackConfiguration))));
