import React from 'react';
import IconTrash from 'images/icon-trash.svg';
import IconTrashWhite from 'images/IconButton/trash-white.svg';
import styles from './DeleteButton.module.scss';
import IconButton, { IconButtonProps } from 'components/basic/IconButton/IconButton';

interface Props extends IconButtonProps {
  white?: boolean;
}

const DeleteButton = (props: Props) => {
  const classNames = [styles['delete-button']];
  const icon = props.white ? IconTrashWhite : IconTrash;

  if (props.className) {
    classNames.push(props.className);
  }

  return (
    <IconButton
      {...props}
      filter={props.white ? false : true}
      tooltipText={props.tooltipText || 'Delete'}
      className={classNames.join(' ')}
      icon={props.icon || icon}
    />
  );
};

export default DeleteButton;
