import React from 'react';
import styles from './ConfirmPopupContent.module.scss';
import { formatError } from 'lib/utils';
import Popup, { PopupProps, PopupState } from 'components/basic/Popup/Popup';
import HighlightedArea from 'components/basic/Popup/HighlightedArea/HighlightedArea';
import Button from 'components/basic/Button/Button';

interface Props extends PopupProps {
  onConfirm: () => Promise<any>;

  confirmButtonText?: string;
  cancelButtonText?: string;
}

interface State extends PopupState {
  loading: boolean;
}

class ConfirmPopupContent extends Popup<Props, State> {
  state: State = {
    loading: false,
  };

  onConfirm = async () => {
    this.setState({ loading: true });

    try {
      await this.props.onConfirm();
      this.close();
      return;
    } catch (err) {
      this.setMessage(formatError(err), 'error');
      this.setState({ loading: false });
    }
  };

  render() {
    let content: React.ReactNode;
    if (typeof this.props.children === 'string') {
      content = <p>{this.props.children}</p>;
    } else {
      content = this.props.children;
    }

    return super.render(
      <div className={styles['confirm-popup-content']}>
        {content}
        <HighlightedArea className="negative-margin">
          <div className={styles.buttons}>
            <Button className={styles.button} onClick={() => this.close()}>
              {this.props.cancelButtonText || 'No'}
            </Button>
            <Button className={styles.button} loading={this.state.loading} onClick={() => this.onConfirm()}>
              {this.props.confirmButtonText || 'Yes'}
            </Button>
          </div>
        </HighlightedArea>
      </div>
    );
  }
}

export default ConfirmPopupContent;
