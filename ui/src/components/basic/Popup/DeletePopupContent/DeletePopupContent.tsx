import React from 'react';
import styles from './DeletePopupContent.module.scss';
import { formatError } from 'lib/utils';
import Popup, { PopupProps, PopupState } from 'components/basic/Popup/Popup';
import HighlightedArea from 'components/basic/Popup/HighlightedArea/HighlightedArea';
import Button from 'components/basic/Button/Button';
import DeleteTextButton from 'components/basic/Button/DeleteTextButton/DeleteTextButton';

interface Props extends PopupProps {
  onDelete: () => Promise<any>;
  deleteButtonText?: string;
  cancelButtonText?: string;

  bottom?: React.ReactNode;
}

interface State extends PopupState {
  loading: boolean;
  error: string;
}

class DeletePopupContent extends Popup<Props, State> {
  state: State = {
    loading: false,
    error: null,
  };

  onDelete = async () => {
    this.setState({ loading: true });

    try {
      await this.props.onDelete();

      this.close();
      return;
    } catch (err) {
      this.setState({ loading: false, error: formatError(err) });
    }
  };

  render() {
    let content: any;

    if (typeof this.props.children === 'string') {
      content = <p>{this.props.children}</p>;
    } else if (typeof this.props.children === 'undefined') {
      content = <p>Do you want to delete this item?</p>;
    } else {
      content = this.props.children;
    }

    return super.render(
      <div className={styles['delete-popup-content']}>
        {content}
        <HighlightedArea
          error={this.state.error && { name: 'Error', message: this.state.error }}
          className={this.props.bottom ? null : 'negative-margin'}
        >
          <div className={styles.buttons}>
            <Button className={styles.cancel} onClick={() => this.close()}>
              {this.props.cancelButtonText || 'Cancel'}
            </Button>
            <DeleteTextButton loading={this.state.loading} onClick={() => this.onDelete()}>
              {this.props.deleteButtonText || 'Delete'}
            </DeleteTextButton>
          </div>
        </HighlightedArea>
        {this.props.bottom}
      </div>
    );
  }
}

export default DeletePopupContent;
