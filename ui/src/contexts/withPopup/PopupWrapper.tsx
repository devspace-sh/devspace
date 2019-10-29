import React from "react";
import { AddExtraProps } from "lib/utils";
import { OpenPopup } from "components/basic/Popup/Popup";
import { bindPopup, PopupContextProvider } from "contexts/withPopup/withPopup";

interface Props {}

interface State {
  popupUUID?: string;
}

export default class PopupWrapper extends React.PureComponent<Props, State> {
  popupQueue: OpenPopup[] = [];
  state: State = {};

  renderPopup() {
    if (!this.state.popupUUID) {
      return null;
    }

    return this.popupQueue.map(popup =>
      AddExtraProps(popup.content, {
        key: popup.uuid,
        display: this.state.popupUUID === popup.uuid,
        close: popup.close
      })
    );
  }

  render() {
    const popupContext = bindPopup(this);

    return (
      <PopupContextProvider value={popupContext}>
        {this.renderPopup()}
        {this.props.children}
      </PopupContextProvider>
    );
  }
}
