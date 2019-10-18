import React from 'react';
import { bindParameter } from 'lib/utils';
import PopupWrapper from 'contexts/withPopup/PopupWrapper';
import AlertPopupContent from 'components/basic/Popup/AlertPopupContent/AlertPopupContent';
import { OpenPopup } from 'components/basic/Popup/Popup';

const reactPopupContext = React.createContext({
  alertPopup: (_title: string, _message: string) => null,
  openPopup: (_: JSX.Element) => null,
  closePopup: () => null,
});

const PopupConsumer: React.ExoticComponent<React.ConsumerProps<Popup>> = reactPopupContext.Consumer;

export interface Popup {
  // Popup function
  alertPopup: (title: string, message: string) => void;
  openPopup: (popup: JSX.Element) => void;
  closePopup: (skipCallback?: boolean) => void;
}

export const PopupContextProvider = reactPopupContext.Provider;

export interface PopupContext {
  popup?: Popup;
}

export default function withPopup<P extends PopupContext>(NewApp: React.ComponentType<P>) {
  return class PopupConsumerComponent extends React.PureComponent<P> {
    render() {
      return <PopupConsumer>{(popup: Popup) => <NewApp popup={popup} {...this.props} />}</PopupConsumer>;
    }
  };
}

export const bindPopup = (app: PopupWrapper): Popup => {
  return {
    alertPopup: bindParameter(alertPopup, app),
    openPopup: bindParameter(openPopup, app),
    closePopup: bindParameter(closePopup, app),
  };
};

const alertPopup = (app: PopupWrapper, title: string, message: string) => {
  openPopup(app, <AlertPopupContent title={title}>{message}</AlertPopupContent>);
};

const openPopup = (app: PopupWrapper, content: JSX.Element) => {
  const newProps: OpenPopup = { content };
  newProps.close = () => nextPopup(app);
  newProps.uuid = Math.random() + '';

  app.popupQueue.push(newProps);
  app.setState({ popupUUID: newProps.uuid });
};

const closePopup = (app: PopupWrapper, skipCallback?: boolean) => {
  if (app.state.popupUUID) {
    if (
      app.popupQueue[app.popupQueue.length - 1].content &&
      app.popupQueue[app.popupQueue.length - 1].content.props &&
      app.popupQueue[app.popupQueue.length - 1].content.props.onClose
    ) {
      if (!skipCallback) {
        app.popupQueue[app.popupQueue.length - 1].content.props.onClose().then(() => {
          nextPopup(app);
        });
      } else {
        nextPopup(app);
      }
    } else {
      app.popupQueue[app.popupQueue.length - 1].close();
    }
  }
};

const nextPopup = (app: PopupWrapper) => {
  let next: string = null;

  app.popupQueue.pop();
  if (app.popupQueue.length > 0) {
    next = app.popupQueue[app.popupQueue.length - 1].uuid;
  }

  app.setState({
    popupUUID: next,
  });
};
