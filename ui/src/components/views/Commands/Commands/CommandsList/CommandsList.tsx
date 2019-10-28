import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext, Command } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import styles from './CommandsList.module.scss';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import LeftAlignIcon from 'images/left-alignment.svg';
import PlayIcon from 'images/play-icon.svg';
import PauseIcon from 'images/pause-blue-icon.svg';
import IconButton from 'components/basic/IconButton/IconButton';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';
import { ApiHostname } from 'lib/rest';
import SimpleCodeLine from 'components/basic/CodeSnippet/SimpleCodeLine/SimpleCodeLine';

interface Props extends DevSpaceConfigContext {
  commandsList: Command[];
  selected: string;
  running: string[];
  onSelect: (commandName: string) => void;
}

interface State {
  openCommandIdx: number;
}

export const getURLByName = (name: string) => {
  if (!name) {
    return null;
  }

  return `ws://${ApiHostname()}/api/command?name=${name}`;
};

class CommandsList extends React.PureComponent<Props, State> {
  state: State = {
    openCommandIdx: -1,
  };

  renderCommands = () => {
    return this.props.devSpaceConfig.rawConfig.commands.map((cmd, idx) => {
      return (
        <PortletSimple key={idx}>
          {{
            top: {
              left: cmd.name,
              right: (
                <React.Fragment>
                  <IconButton
                    filter={false}
                    icon={LeftAlignIcon}
                    tooltipText="Show Command"
                    onClick={() => {
                      this.onShowCommandClick(idx);
                    }}
                  />
                  <IconButton
                    filter={false}
                    icon={this.props.running.find((url) => url === getURLByName(cmd.name)) ? PauseIcon : PlayIcon}
                    onClick={() => this.props.onSelect(cmd.name)}
                  />
                </React.Fragment>
              ),
            },
            content:
              idx === this.state.openCommandIdx ? (
                <div className={styles['show-command']}>
                  <CodeSnippet className={styles.codesnippet}>
                    <SimpleCodeLine>{cmd.command}</SimpleCodeLine>
                  </CodeSnippet>
                </div>
              ) : null,
          }}
        </PortletSimple>
      );
    });
  };

  onShowCommandClick = (idx: number) => {
    this.setState({ openCommandIdx: this.state.openCommandIdx === idx ? -1 : idx });
  };

  render() {
    return (
      <div className={styles['commands-list']}>
        <div className={styles['commands-list-wrapper']}>{this.renderCommands()}</div>
      </div>
    );
  }
}

export default withDevSpaceConfig(CommandsList);
