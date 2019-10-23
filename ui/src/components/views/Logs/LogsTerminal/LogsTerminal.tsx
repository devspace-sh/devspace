import React from 'react';
import { Terminal } from 'xterm';
import { AttachAddon } from 'lib/attach';
import style from './LogsTerminal.module.scss';
import Button from 'components/basic/Button/Button';

export interface LogsTerminalProps {
  className?: string;
  url: string;
  interactive: boolean;
  show: boolean;
  onClose?: () => void;
}

interface State {
  fullscreen: boolean;
}

const MINIMUM_COLS = 2;
const MINIMUM_ROWS = 1;

class LogsTerminal extends React.PureComponent<LogsTerminalProps, State> {
  state: State = {
    fullscreen: false,
  };

  socket: WebSocket;
  term: Terminal;

  ref: HTMLDivElement;
  initialWidth: number;
  initialHeight: number;

  updateDimensions = () => {
    if (this.ref.children && this.ref.children.length > 0) {
      (this.ref.children[0] as any).style.display = 'none';
    }

    const computedStyle = window.getComputedStyle(this.ref);
    this.initialHeight = parseInt(computedStyle.getPropertyValue('height'));
    this.initialWidth = Math.max(0, parseInt(computedStyle.getPropertyValue('width')));

    if (this.ref.children && this.ref.children.length > 0) {
      (this.ref.children[0] as any).style.display = 'block';
    }

    this.fit();
  };

  fit = () => {
    if (this.term) {
      // Force a full render
      const core = (this.term as any)._core;
      const availableHeight = this.initialHeight;
      const availableWidth = this.initialWidth - core.viewport.scrollBarWidth;
      const dims = {
        cols: Math.max(MINIMUM_COLS, Math.floor(availableWidth / core._renderService.dimensions.actualCellWidth)),
        rows: Math.max(MINIMUM_ROWS, Math.floor(availableHeight / core._renderService.dimensions.actualCellHeight)),
      };
      if (this.term.rows !== dims.rows || this.term.cols !== dims.cols) {
        core._renderService.clear();
        this.term.resize(Math.floor(dims.cols), Math.floor(dims.rows));
      }
    }
  };

  attach(ref: HTMLDivElement) {
    if (!ref || this.term) {
      return;
    }

    this.ref = ref;
    this.updateDimensions();

    this.term = new Terminal({
      // We need this setting to automatically convert \n -> \r\n
      convertEol: true,
      disableStdin: !this.props.interactive,
    });

    // Open the websocket
    this.socket = new WebSocket(this.props.url);
    const attachAddon = new AttachAddon(this.socket, {
      bidirectional: this.props.interactive,
      onClose: this.props.onClose,
      onError: (err) => {
        this.term.writeln('\u001b[31m' + err.message);
      },
    });

    // Attach the socket to term
    this.term.open(ref);
    this.term.loadAddon(attachAddon);
    this.fit();

    window.addEventListener('resize', this.updateDimensions, true);
  }

  componentWillUnmount() {
    if (this.socket) {
      this.socket.close();
    }

    window.removeEventListener('resize', this.updateDimensions);
  }

  render() {
    const classnames = [style['terminal-wrapper']];
    if (this.props.className) {
      classnames.push(this.props.className);
    }
    if (this.state.fullscreen) {
      classnames.push(style['fullscreen']);
    }

    return (
      <div className={classnames.join(' ')} style={{ display: this.props.show ? 'flex' : 'none' }}>
        <Button onClick={() => this.setState({ fullscreen: !this.state.fullscreen }, this.updateDimensions)}>
          Fullscreen
        </Button>
        <div className={style['terminal']} ref={(ref) => this.attach(ref)} />
      </div>
    );
  }
}

export default LogsTerminal;
