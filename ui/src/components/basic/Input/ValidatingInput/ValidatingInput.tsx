import React from 'react';
import Input, { InputProps } from '../Input';
import styles from './ValidatingInput.module.scss';

interface Props extends InputProps {
  valid: boolean;
}

const ValidatingInput = (props: Props) => {
  return (
    <Input
      className={props.valid ? styles['validating-input'] + ' ' + styles.valid : styles['validating-input']}
      {...props}
    />
  );
};

export default ValidatingInput;
