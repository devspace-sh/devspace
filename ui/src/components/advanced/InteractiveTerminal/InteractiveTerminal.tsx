import React from 'react';
import { Terminal } from '@xterm/xterm';
import { AttachAddon } from '@xterm/addon-attach';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
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

class InteractiveTerminal extends React.PureComponent<InteractiveTerminalProps, State> {
  state: State = {
    fullscreen: false,
  };

  private resizeId: string = uuidv4();
  private socket: WebSocket;
  private term: Terminal;
  private attachAddon: AttachAddon;
  private fitAddon: FitAddon;

  private needUpdate: boolean;
  private closed: boolean = false;

  updateDimensions = () => {
    if (!this.props.show) {
      this.needUpdate = true;
      return;
    }

    this.fit();
  };

  fit = () => {
    if (this.term && this.fitAddon && this.props.show) {
      try {
        this.fitAddon.fit();
      } catch (err) {
        console.error(err);
        return;
      }

      // send dims to the server
      if (this.props.remoteResize && this.socket && this.socket.readyState === WebSocket.OPEN) {
        authFetch(`/api/resize?resize_id=${this.resizeId}&width=${this.term.cols}&height=${this.term.rows}`).catch(err => {
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

    this.term = new Terminal({
      // We need this setting to automatically convert \n -> \r\n
      convertEol: true,
      disableStdin: !this.props.interactive,
      theme: {
        background: '#263544',
        foreground: '#AFC6D2',
      },
    });
    this.fitAddon = new FitAddon();

    // Open the websocket
    this.socket = new WebSocket(this.props.url + (this.props.remoteResize ? "&resize_id="+this.resizeId : ""));
    if (this.props.remoteResize) {
      this.socket.addEventListener('open', this.fit);
    }
    this.socket.addEventListener('close', this.handleSocketClose);
    this.socket.addEventListener('error', this.handleSocketError);
    this.attachAddon = new AttachAddon(this.socket, {
      bidirectional: this.props.interactive,
    });

    this.term.open(ref);
    this.term.loadAddon(this.fitAddon);
    this.term.loadAddon(this.attachAddon);
    this.fit();

    window.addEventListener('resize', this.updateDimensions, true);
  }

  handleSocketClose = (event: CloseEvent) => {
    if (this.closed) {
      return;
    }

    if (event.code === 1011 && event.reason) {
      this.term.writeln('\u001b[31m' + event.reason);
    }

    if (this.props.closeDelay && this.props.closeOnConnectionLost && this.props.onClose) {
      this.term.writeln(`Connection closed, will close in ${this.props.closeDelay / 1000} seconds`);
      setTimeout(this.props.onClose, this.props.closeDelay);
      return;
    }

    this.term.writeln('Connection closed');
    if (this.props.closeOnConnectionLost && this.props.onClose) {
      this.props.onClose();
    }
  };

  handleSocketError = () => {
    if (!this.closed) {
      this.term.writeln('\u001b[31mConnection error');
    }
  };

  componentDidUpdate() {
    if (this.props.show && this.needUpdate) {
      this.updateDimensions();
    }
  }

  componentWillUnmount() {
    this.closed = true;
    window.removeEventListener('resize', this.updateDimensions, true);
    if (this.socket) {
      this.socket.removeEventListener('open', this.fit);
      this.socket.removeEventListener('close', this.handleSocketClose);
      this.socket.removeEventListener('error', this.handleSocketError);
      this.socket.close();
    }
    if (this.attachAddon) {
      this.attachAddon.dispose();
    }
    if (this.fitAddon) {
      this.fitAddon.dispose();
    }
    if (this.term) {
      this.term.dispose();
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
