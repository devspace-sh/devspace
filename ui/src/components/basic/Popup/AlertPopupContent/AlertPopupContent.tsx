import React, { ReactNode } from 'react';
import styles from './AlertPopupContent.module.scss';
import { formatError } from 'lib/utils';
import Popup, { PopupProps, PopupState } from 'components/basic/Popup/Popup';
import Button from 'components/basic/Button/Button';

interface Props extends PopupProps {
  onConfirm?: () => Promise<any>;
  buttonText?: string;
  className?: string;
  hideCloseButton?: boolean;
}

interface State extends PopupState {
  loading: boolean;
}

class AlertPopupContent extends Popup<Props, State> {
  state: State = {
    loading: false,
  };

  onConfirm = async () => {
    if (this.props.onConfirm) {
      this.setState({ loading: true });

      try {
        await this.props.onConfirm();
        this.close();
      } catch (err) {
        this.setMessage(formatError(err), 'error');
        this.setState({ loading: false });
      }
    } else {
      this.close();
    }
  };

  render() {
    let content: ReactNode;
    if (typeof this.props.children === 'string') {
      content = <p>{this.props.children}</p>;
    } else {
      content = this.props.children;
    }

    return super.render(
      <div
        className={
          this.props.className ? styles['alert-popup-content'] + ' ' + this.props.className : styles['alert-popup-content']
        }
      >
        {content}
        {!this.props.hideCloseButton && (
          <div className={styles.buttons}>
            <Button loading={this.state.loading} onClick={() => this.onConfirm()}>
              {this.props.buttonText || 'Close'}
            </Button>
          </div>
        )}
      </div>
    );
  }
}

export default AlertPopupContent;
