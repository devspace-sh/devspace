import React from 'react';
import IconClose from 'images/icon-close.svg';
import IconCloseWhite from 'images/close-white.svg';
import styles from './CloseButton.module.scss';
import IconButton, { IconButtonProps } from 'components/basic/IconButton/IconButton';

interface Props extends IconButtonProps {
  white?: boolean;
}

const CloseButton = (props: Props) => {
  return (
    <IconButton
      {...props}
      className={props.className ? styles['close-button'] + ' ' + props.className : styles['close-button']}
      icon={props.white ? IconCloseWhite : IconClose}
    />
  );
};

export default CloseButton;
