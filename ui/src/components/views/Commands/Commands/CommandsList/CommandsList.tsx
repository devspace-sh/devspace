import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext, Command } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import styles from './CommandsList.module.scss';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import LeftAlignIcon from 'images/left-alignment.svg';
import PlayIcon from 'images/play-icon.svg';
import IconButton from 'components/basic/IconButton/IconButton';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';

interface Props extends DevSpaceConfigContext {
  commandsList: Command[];
}

interface State {
  openCommandIdx: number;
}

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
                  <IconButton filter={false} icon={PlayIcon} />
                </React.Fragment>
              ),
            },
            content:
              idx === this.state.openCommandIdx ? (
                <div className={styles['show-command']}>
                  <CodeSnippet>{cmd.command}</CodeSnippet>
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
    return <div className={styles['commands-list']}>{this.renderCommands()}</div>;
  }
}

export default withDevSpaceConfig(CommandsList);
