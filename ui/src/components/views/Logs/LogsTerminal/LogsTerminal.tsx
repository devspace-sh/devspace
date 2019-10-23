import React from 'react';
import { Terminal } from 'xterm';
import { AttachAddon } from 'lib/attach';

export interface LogsTerminalProps {
  url: string;
  interactive: boolean;
  show: boolean;
  onClose?: () => void;
}

interface State {}

class LogsTerminal extends React.PureComponent<LogsTerminalProps, State> {
  socket: WebSocket;
  term: Terminal;

  attach(ref: HTMLDivElement) {
    if (!ref || this.term) {
      return;
    }

    this.term = new Terminal({
      // We need this setting to automatically convert \n -> \r\n
      convertEol: true,
      disableStdin: !this.props.interactive,
    });

    // Open the websocket
    this.socket = new WebSocket(this.props.url);
    const attachAddon = new AttachAddon(this.socket, { bidirectional: this.props.interactive, onClose: this.props.onClose });

    // Attach the socket to term
    this.term.open(ref);
    this.term.loadAddon(attachAddon);
  }

  componentWillUnmount() {
    if (this.socket) {
      this.socket.close();
    }
  }

  render() {
    return <div style={{ display: this.props.show ? 'block' : 'none' }} ref={(ref) => this.attach(ref)} />;
  }
}

export default LogsTerminal;
