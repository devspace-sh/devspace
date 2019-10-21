import React from 'react';
import { Terminal } from 'xterm';
import { AttachAddon } from 'lib/attach';
import { ApiHostname } from 'lib/rest';

export interface LogsTerminalProps {
  pod: string;
  container: string;
  namespace: string;

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
      disableStdin: true,
    });

    // Open the websocket
    this.socket = new WebSocket(
      `ws://${ApiHostname()}/api/logs?namespace=${this.props.namespace}&name=${this.props.pod}&container=${
        this.props.container
      }`
    );
    const attachAddon = new AttachAddon(this.socket, { bidirectional: false, onClose: this.props.onClose });

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
