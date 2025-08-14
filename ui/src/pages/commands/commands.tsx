import React from 'react';
import { withRouter, RouteComponentProps } from 'react-router';
import styles from './commands.module.scss';
import PageLayout from 'components/basic/PageLayout/PageLayout';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import { V1PodList } from '@kubernetes/client-node';
import withDevSpaceConfig, { Command, DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withWarning, { WarningContext } from 'contexts/withWarning/withWarning';
import CommandsLinkTabSelector from 'components/basic/LinkTabSelector/CommandsLinkTabSelector/CommandsLinkTabSelector';
import CommandsList, { getURLByName } from 'components/views/Commands/Commands/CommandsList/CommandsList';
import InteractiveTerminal, { InteractiveTerminalProps } from 'components/advanced/InteractiveTerminal/InteractiveTerminal';
import AdvancedCodeLine from 'components/basic/CodeSnippet/AdvancedCodeLine/AdvancedCodeLine';
import Button from '../../components/basic/Button/Button';

interface Props extends DevSpaceConfigContext, PopupContext, WarningContext, RouteComponentProps {}

interface State {
  podList?: V1PodList;
  selected?: string;
  terminals: StateTerminalProps[];
  showInternal: boolean;
}

interface StateTerminalProps extends InteractiveTerminalProps {
  name: string;
}

class Commands extends React.PureComponent<Props, State> {
  state: State = {
    terminals: [],
    showInternal: false
  };

  onSelectCommand = (commandName: string) => {
    // Check if terminal already exists
    const terminalURL = getURLByName(commandName);
    const idx = this.state.terminals.findIndex((t) => t.url === terminalURL);

    if (idx === -1) {
      this.setState({
        selected: terminalURL,
        terminals: [
          {
            name: commandName,
            url: terminalURL,
          },
        ],
      });
    } else {
      const newTerminals = [...this.state.terminals];
      newTerminals.splice(idx, 1);
      this.setState({
        terminals: newTerminals,
        selected: null,
      });
    }
  };

  renderTerminals = () => {
    if (!this.state.selected) {
      return <div className={styles['nothing-selected']}>Please start a command on the right side</div>;
    }

    return this.state.terminals.map((terminal) => (
      <InteractiveTerminal
        key={terminal.url}
        {...terminal}
        show={this.state.selected === terminal.url}
        interactive={true}
        firstLine={<AdvancedCodeLine>devspace run {terminal.name}</AdvancedCodeLine>}
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

  getVisibleCommands(commands:  { [key: string]: Command }) {
    if (this.state.showInternal === false) {
      return Object.fromEntries(Object.entries(commands).filter(([_key, config]) => {
        return config.internal !== true
      }))
    }
    return commands;
  }

  render() {
    return (
      <PageLayout className={styles['commands-component']} heading={<CommandsLinkTabSelector />}>
        {!this.props.devSpaceConfig.config ||
          !this.props.devSpaceConfig.config.commands ||
          Object.entries(this.props.devSpaceConfig.config.commands).length === 0 ? (
          <div className={styles['no-config']}>
            <div>
              No commands available. Take a look at&nbsp;
              <a target="_blank" href="https://devspace.cloud/docs/cli/configuration/custom-commands">
                commands
              </a>
              &nbsp;to add commands to your config
            </div>
          </div>
        ) : (
          <React.Fragment>
            {this.renderTerminals()}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              <div style={{ height: '3rem', display: 'flex', justifyContent: 'right', marginRight: '10px', marginBottom: '10px' }}>
                <Button
                  onClick={() => {
                    this.setState((state) => {
                      return {
                        showInternal: !state.showInternal
                      }
                    })
                  }}
                >{this.state.showInternal ? 'Hide internal' : 'Show internal'}</Button>
              </div>

              <div className={styles['info-part']} style={{ overflowY: 'auto' }}>
                <CommandsList
                  commandsList={this.getVisibleCommands(this.props.devSpaceConfig.config.commands)}
                  running={this.state.terminals.map((terminal) => terminal.url)}
                  selected={this.state.selected}
                  onSelect={this.onSelectCommand}
                />
              </div>
            </div>
          </React.Fragment>
        )}
      </PageLayout>
    );
  }
}

export default withRouter(withPopup(withDevSpaceConfig(withWarning(Commands))));
