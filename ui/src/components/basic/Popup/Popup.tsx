import React, { ReactNode } from 'react';
import styles from './Popup.module.scss';
import CloseButton from 'components/basic/IconButton/CloseButton/CloseButton';

export interface OpenPopup {
  uuid?: string;
  display?: boolean;

  content?: JSX.Element;

  // We need this function so that the popup knows
  // how to close itself
  close?: () => void;
}

export interface PopupProps extends OpenPopup {
  // Title
  title?: string;

  // Close button
  onClose?: () => Promise<boolean>;

  // Is popup closeable?
  closable?: boolean;

  // Classname
  className?: string;

  // Custom component
  children?: ReactNode;

  showOverflow?: boolean;

  width?: string;
}

export interface PopupState {
  // Status message below buttons
  statusMessage?: string;
  statusMessageClassname?: string;
}

class Popup<P extends PopupProps, S extends PopupState> extends React.PureComponent<P, S> {
  setMessage(message: string, classname: string) {
    this.setState({
      statusMessage: message,
      statusMessageClassname: classname,
    });
  }

  // Overwrite on close
  onClose = (e?: React.MouseEvent<any, MouseEvent>) => {
    e.stopPropagation();
    this.close();
  };

  close = () => {
    if (this.props.onClose) {
      this.props.onClose().then((shouldClose) => {
        if (shouldClose) {
          this.props.close();
        }
      });

      return;
    }

    this.props.close();
  };

  render(children?: ReactNode, className?: string) {
    let classNames = styles.popup;
    if (this.props.className) {
      classNames += ' ' + this.props.className;
    }
    if (className) {
      classNames += ' ' + className;
    }

    let style: React.CSSProperties = {};
    if (!this.props.display) {
      style = { display: 'none' };
    }
    const popupStyle: React.CSSProperties = {};
    if (this.props.width) {
      popupStyle.width = this.props.width;
    }

    return (
      <div
        className={styles['popup-container']}
        onMouseDown={this.props.closable !== false ? this.onClose : undefined}
        style={style}
      >
        <div style={popupStyle} className={classNames} onMouseDown={(e) => e.stopPropagation()}>
          {this.props.closable !== false && <CloseButton className={styles['popup-close']} onClick={this.onClose} />}
          {this.props.title ? (
            <div className={styles.title}>
              <h3>{this.props.title}</h3>
              <hr />
            </div>
          ) : null}
          {children || this.props.children}
          {this.state && this.state.statusMessage ? (
            <div
              className={
                this.state.statusMessageClassname
                  ? styles['status-message'] + ' ' + this.state.statusMessageClassname
                  : styles['status-message']
              }
            >
              {this.state.statusMessage}
            </div>
          ) : null}
        </div>
      </div>
    );
  }
}

export default Popup;
