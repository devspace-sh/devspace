import React from 'react';
import { bindParameter } from 'lib/utils';
import WarningWrapper from './WarningWrapper';
import { WarningProps } from 'components/basic/Warning/Warning';

const reactWarningContext = React.createContext({
  show: (_message: React.ReactNode) => null,
  getActive: () => null,
  close: () => null,
});

const WarningConsumer: React.ExoticComponent<React.ConsumerProps<Warning>> = reactWarningContext.Consumer;

export interface Warning {
  show: (message: React.ReactNode) => void;
  getActive: () => WarningProps;
  close: () => void;
}

export const WarningContextProvider = reactWarningContext.Provider;

export interface WarningContext {
  warning?: Warning;
}

export default function withWarning<P extends WarningContext>(NewApp: React.ComponentType<P>) {
  return class WarningConsumerComponent extends React.PureComponent<P> {
    render() {
      return <WarningConsumer>{(warning: Warning) => <NewApp warning={warning} {...this.props} />}</WarningConsumer>;
    }
  };
}

export const bindWarning = (app: WarningWrapper): Warning => {
  return {
    show: bindParameter(show, app),
    getActive: bindParameter(getActive, app),
    close: bindParameter(close, app),
  };
};

const show = (app: WarningWrapper, content: JSX.Element) => {
  const newProps: WarningProps = {
    uuid: Math.random() + '',
    children: content,
    close: () => nextPopup(app),
  };

  app.warningQueue.push(newProps);
  app.setState({ warningUUID: newProps.uuid });
};

const getActive = (app: WarningWrapper) => {
  if (!app.warningQueue.length) {
    return null;
  }

  return app.warningQueue[app.warningQueue.length - 1];
};

const close = (app: WarningWrapper) => {
  if (app.state.warningUUID) {
    app.warningQueue[app.warningQueue.length - 1].close();
  }
};

const nextPopup = (app: WarningWrapper) => {
  let next: string = null;

  app.warningQueue.pop();
  if (app.warningQueue.length > 0) {
    next = app.warningQueue[app.warningQueue.length - 1].uuid;
  }

  app.setState({
    warningUUID: next,
  });
};
