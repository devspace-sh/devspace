import React from 'react';
import { Terminal } from 'xterm';
import { AttachAddon } from 'xterm-addon-attach';

interface Props {}
interface State {}

class LogsTerminal extends React.PureComponent<Props, State> {
  attach(ref: HTMLDivElement) {
    if (!ref) {
      return;
    }

    const term = new Terminal();
    const socket = new WebSocket(
      'ws://localhost:8090/api/logs?namespace=test&name=quickstart-6c76fbc6f4-46nfd&container=container-0'
    );
    const attachAddon = new AttachAddon(socket, { bidirectional: false });

    // Attach the socket to term
    term.open(ref);
    term.loadAddon(attachAddon);
  }

  render() {
    return <div ref={(ref) => this.attach(ref)} />;
  }
}

export default LogsTerminal;
