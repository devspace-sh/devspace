import React from 'react';
import { Terminal } from 'xterm';
import { AttachAddon } from 'lib/attach';
import styles from './InteractiveTerminal.module.scss';
import MaximizeButton from 'components/basic/IconButton/MaximizeButton/MaximizeButton';
import IconButton from 'components/basic/IconButton/IconButton';
import IconTrash from 'images/trash.svg';
import {uuidv4} from "../../../lib/utils";
import authFetch from "../../../lib/fetch";

export interface InteractiveTerminalProps {
  className?: string;
  url: string;
  interactive?: boolean;
  show?: boolean;

  firstLine?: React.ReactNode;

  closeOnConnectionLost?: boolean;
  closeDelay?: number;

  remoteResize?: boolean;
  onClose?: () => void;
}

interface State {
  fullscreen: boolean;
}

const MINIMUM_COLS = 2;
const MINIMUM_ROWS = 1;

class InteractiveTerminal extends React.PureComponent<InteractiveTerminalProps, State> {
  state: State = {
    fullscreen: false,
  };

  private resizeId: string = uuidv4();
  private socket: WebSocket;
  private term: Terminal;

  private ref: HTMLDivElement;
  private needUpdate: boolean;
  private initialWidth: number;
  private initialHeight: number;
  private closed: boolean = false;

  updateDimensions = () => {
    if (!this.props.show) {
      this.needUpdate = true;
      return;
    }

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
    if (this.term && this.props.show) {
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

      // send dims to the server
      if (this.props.remoteResize) {
        authFetch(`/api/resize?resize_id=${this.resizeId}&width=${dims.cols}&height=${dims.rows}`).catch(err => {
          console.error(err);
        })
      }

      this.needUpdate = false;
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
      theme: {
        background: '#263544',
        foreground: '#AFC6D2',
      },
    });

    // Open the websocket
    this.socket = new WebSocket(this.props.url + (this.props.remoteResize ? "&resize_id="+this.resizeId : ""));
    const attachAddon = new AttachAddon(this.socket, {
      bidirectional: this.props.interactive,
      onClose: () => {
        if (!this.closed) {
          if (this.props.closeDelay && this.props.closeOnConnectionLost && this.props.onClose) {
            this.term.writeln(`Connection closed, will close in ${this.props.closeDelay / 1000} seconds`);
            setTimeout(this.props.onClose, this.props.closeDelay);
            return;
          }

          this.term.writeln('Connection closed');
          if (this.props.closeOnConnectionLost && this.props.onClose) {
            this.props.onClose();
          }
        }
      },
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

  componentDidUpdate() {
    if (this.props.show && this.needUpdate) {
      this.updateDimensions();
    }
  }

  componentWillUnmount() {
    this.closed = true;
    window.removeEventListener('resize', this.updateDimensions, true);
    if (this.socket) {
      this.socket.close();
    }
  }

  render() {
    const classnames = [styles['terminal-wrapper']];
    if (this.props.className) {
      classnames.push(this.props.className);
    }
    if (this.state.fullscreen) {
      classnames.push(styles['fullscreen']);
    }

    return (
      <div className={classnames.join(' ')} style={{ display: this.props.show ? 'flex' : 'none' }}>
        <div className={styles.header}>
          {this.props.firstLine || <div />}
          <div className={styles.buttons}>
            <MaximizeButton
              maximized={this.state.fullscreen}
              className={styles.maximize}
              filter={false}
              tooltipPosition={'bottom'}
              onClick={() => this.setState({ fullscreen: !this.state.fullscreen }, this.updateDimensions)}
            />
            <IconButton
              icon={IconTrash}
              filter={false}
              tooltipText="Kill Terminal"
              tooltipPosition={'bottom'}
              onClick={() => {
                this.socket.close();
                if (this.props.onClose) {
                  this.props.onClose();
                }
              }}
            />
          </div>
        </div>
        <div className={styles['terminal']} ref={(ref) => this.attach(ref)} />
      </div>
    );
  }
}

export default InteractiveTerminal;
