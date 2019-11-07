import * as React from 'react';
import styles from './ErrorMessage.module.scss';
import { formatError } from 'lib/utils';

interface Props {
  className?: string;
  children: React.ReactNode;
}

export default function(props: Props) {
  return (
    <div
      className={
        props.className ? styles['error-component-wrapper'] + ' ' + props.className : styles['error-component-wrapper']
      }
    >
      {formatError(props.children)}
    </div>
  );
}
