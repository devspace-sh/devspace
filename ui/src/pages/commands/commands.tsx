import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './commands.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import { V1PodList } from '@kubernetes/client-node';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';
import CommandsLinkTabSelector from 'components/basic/LinkTabSelector/CommandsLinkTabSelector/CommandsLinkTabSelector';
import CommandsList, { getURLByName } from 'components/views/Commands/Commands/CommandsList/CommandsList';
import InteractiveTerminal, { InteractiveTerminalProps } from 'components/advanced/InteractiveTerminal/InteractiveTerminal';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: string;
  terminals: InteractiveTerminalProps[];
}

class Commands extends React.PureComponent<Props, State> {
  state: State = {
    terminals: [],
  };

  onSelectCommand = (commandName: string) => {
    // Check if terminal already exists
    const terminalURL = getURLByName(commandName);
    const idx = this.state.terminals.findIndex((t) => t.url === terminalURL);

    if (idx === -1) {
      this.setState({
        selected: terminalURL,
        terminals: [
          ...this.state.terminals,
          {
            url: terminalURL,
          },
        ],
      });
    } else {
      if (this.state.selected === terminalURL) {
        const newTerminals = [...this.state.terminals];
        newTerminals.splice(idx, 1);
        this.setState({
          terminals: newTerminals,
          selected: null,
        });
      } else {
        this.setState({
          selected: terminalURL,
        });
      }
    }
  };

  renderTerminals = () => {
    return this.state.terminals.map((terminal) => (
      <InteractiveTerminal
        key={terminal.url}
        {...terminal}
        show={this.state.selected === terminal.url}
        interactive={true}
        onClose={() => {
          const newTerminals = [...this.state.terminals];
          const idx = this.state.terminals.findIndex((t) => t.url === terminal.url);

          if (idx !== -1) {
            newTerminals.splice(idx, 1);
            this.setState({
              terminals: newTerminals,
            });
          }
        }}
      />
    ));
  };

  render() {
    return (
      <PageLayout className={styles['commands-component']} heading={<CommandsLinkTabSelector />}>
        {!this.props.devSpaceConfig.rawConfig ||
        !this.props.devSpaceConfig.rawConfig.commands ||
        this.props.devSpaceConfig.rawConfig.commands.length === 0 ? (
          <div className={styles['no-config']}>There is no command available</div>
        ) : (
          <React.Fragment>
            {this.renderTerminals()}
            <div className={styles['info-part']}>
              <CommandsList
                commandsList={this.props.devSpaceConfig.rawConfig.commands}
                running={this.state.terminals.map((terminal) => terminal.url)}
                selected={this.state.selected}
                onSelect={this.onSelectCommand}
              />
            </div>
          </React.Fragment>
        )}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(Commands))));
