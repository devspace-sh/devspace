import React from 'react';
import styles from './IconButton.module.scss';
import ButtonBase from 'components/basic/ButtonBase/ButtonBase';
import Tooltip from 'components/basic/Tooltip/Tooltip';

export interface IconButtonProps {
  tooltipText?: string;
  tooltipPosition?: 'top' | 'bottom' | 'left' | 'right' | 'for-icon';

  // Here we keep track of special class names that we can apply
  className?: 'filter' | string;
  style?: React.CSSProperties;

  loading?: boolean;
  icon?: string;

  onClick?: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;

  filter?: boolean;
  type?: 'button' | 'submit' | 'reset';
}

export default function IconButton(props: IconButtonProps) {
  const className = [styles['icon-button']];

  if (props.filter === undefined || props.filter === true) className.push(styles.filter);
  if (props.className) className.push(props.className);

  const button = <ButtonBase {...props} className={className.join(' ')} />;

  return props.tooltipText ? (
    <Tooltip
      className={props.className ? props.className + ' ' + styles['icon-button-tooltip'] : styles['icon-button-tooltip']}
      text={props.tooltipText}
      position={props.tooltipPosition || 'top'}
    >
      {button}
    </Tooltip>
  ) : (
    button
  );
}
