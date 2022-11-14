import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext, Command } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import styles from './CommandsList.module.scss';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import LeftAlignIcon from 'images/left-alignment.svg';
import PlayIcon from 'images/play-icon.svg';
import PauseIcon from 'images/pause-blue-icon.svg';
import IconButton from 'components/basic/IconButton/IconButton';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';
import { ApiHostname, ApiWebsocketProtocol } from 'lib/rest';
import SimpleCodeLine from 'components/basic/CodeSnippet/SimpleCodeLine/SimpleCodeLine';

interface Props extends DevSpaceConfigContext {
  commandsList: { [key: string]: Command };
  selected: string;
  running: string[];
  onSelect: (commandName: string) => void;
}

interface State {
  openCommandKey: string;
}

export const getURLByName = (name: string) => {
  if (!name) {
    return null;
  }

  return `${ApiWebsocketProtocol()}://${ApiHostname()}/api/command?name=${name}`;
};

class CommandsList extends React.PureComponent<Props, State> {
  state: State = {
    openCommandKey: "",
  };

  renderCommands = () => {
    return Object.entries(this.props.commandsList).map(([key, cmd]) => {
      return <PortletSimple key={key}>
        {{
          top: {
            left: key,
            right: (
                <React.Fragment>
                  <IconButton
                      filter={false}
                      icon={LeftAlignIcon}
                      tooltipText="Show Command"
                      onClick={() => {
                        this.onShowCommandClick(key);
                      }}
                  />
                  <IconButton
                      filter={false}
                      icon={this.props.running.find((url) => url === getURLByName(key)) ? PauseIcon : PlayIcon}
                      onClick={() => this.props.onSelect(key)}
                  />
                </React.Fragment>
            ),
          },
          content:
              key === this.state.openCommandKey ? (
                  <div className={styles['show-command']}>
                    <CodeSnippet className={styles.codesnippet}>
                      <SimpleCodeLine>{cmd.command}</SimpleCodeLine>
                    </CodeSnippet>
                  </div>
              ) : null,
        }}
      </PortletSimple>
    })
  };

  onShowCommandClick = (key: string) => {
    this.setState({ openCommandKey: this.state.openCommandKey === key ? "" : key });
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
