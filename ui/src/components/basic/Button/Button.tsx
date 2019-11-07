import * as React from 'react';
import styles from './Button.module.scss';
import ButtonBase, { ButtonBaseProps } from 'components/basic/ButtonBase/ButtonBase';

export interface ButtonProps extends ButtonBaseProps {}

export default function Button(props: ButtonProps) {
  const className = props.className ? styles.button + ' ' + props.className : styles.button;

  return (
    <ButtonBase {...props} className={className}>
      {props.children}
    </ButtonBase>
  );
}
