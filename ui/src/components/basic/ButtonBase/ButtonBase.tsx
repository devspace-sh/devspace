import React from 'react';
import styles from './ButtonBase.module.scss';

export interface ButtonBaseProps {
  className?: string;
  style?: React.CSSProperties;
  type?: 'button' | 'submit' | 'reset';

  loading?: boolean;
  faIcon?: React.ReactNode;
  icon?: string;

  onClick?: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
  children?: React.ReactNode;
}

const renderIcon = (props: ButtonBaseProps) => {
  const classNames = ['icon-container'];
  if (props.children) {
    classNames.push(styles['with-border']);
  }

  // Show loading icon if loading
  if (props.loading) {
    classNames.push(styles['button-loading-wrapper']);

    return (
      <span className={classNames.join(' ')}>
        <span className={styles['button-loading-circle']} />
      </span>
    );
  }

  // Show svg icon if defined
  if (props.icon) {
    return (
      <span className={classNames.join(' ')}>
        <img className="img-icon" src={props.icon} />
      </span>
    );
  }

  // Show fa icon if defined
  if (props.faIcon) {
    return <span className={classNames.join(' ')}>{props.faIcon}</span>;
  }

  // Text button only
  return null;
};

export default function ButtonBase(props: ButtonBaseProps) {
  const classNames: string[] = [styles['button-base']];
  if (props.className) {
    classNames.push(props.className);
  }
  if (props.loading) {
    classNames.push(styles['-loading']);
  }

  return (
    <button
      onClick={!props.loading ? props.onClick : undefined}
      className={classNames.join(' ')}
      style={props.style}
      type={props.type || 'button'}
    >
      {renderIcon(props)}
      {props.children && <span className={'text'}>{props.children}</span>}
    </button>
  );
}
