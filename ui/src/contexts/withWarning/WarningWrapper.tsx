import React from 'react';
import Warning, { WarningProps } from 'components/basic/Warning/Warning';
import { bindWarning, WarningContextProvider } from './withWarning';

interface Props {}

interface State {
  warningUUID: string;
}

export default class WarningWrapper extends React.PureComponent<Props, State> {
  warningQueue: WarningProps[] = [];
  state: State = {
    warningUUID: null,
  };

  renderWarning() {
    if (!this.state.warningUUID) {
      return null;
    }

    return this.warningQueue.map((warning) => (
      <Warning key={warning.uuid} {...warning} show={this.state.warningUUID === warning.uuid} />
    ));
  }

  render() {
    return (
      <WarningContextProvider value={bindWarning(this)}>
        {this.renderWarning()}
        {this.props.children}
      </WarningContextProvider>
    );
  }
}
