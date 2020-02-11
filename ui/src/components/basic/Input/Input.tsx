import React from 'react';
import styles from './Input.module.scss';
import CopyButton from 'components/basic/IconButton/CopyButton/CopyButton';

export interface InputProps {
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  style?: React.CSSProperties;
  value?: string | number;
  showCopyButton?: boolean;
  name?: string;
  placeholder?: string;
  type?: 'text' | 'number' | 'password' | 'email';
  className?: string;
  disabled?: boolean;
}

const Input = (props: InputProps) => {
  const input = (
    <input
      style={props.style}
      readOnly={!props.onChange}
      className={props.className ? styles['input-component'] + ' ' + props.className : styles['input-component']}
      type={props.type ? props.type : 'text'}
      placeholder={props.placeholder}
      name={props.name}
      onChange={(e) => props.onChange(e)}
      value={props.value}
      onFocus={props.onFocus}
      onBlur={props.onBlur}
      disabled={props.disabled}
    />
  );

  if (props.showCopyButton) {
    return (
      <span className={styles['input-component-container']}>
        {input}
        <CopyButton textToCopy={props.value as string} />
      </span>
    );
  } else {
    return input;
  }
};

export default Input;
