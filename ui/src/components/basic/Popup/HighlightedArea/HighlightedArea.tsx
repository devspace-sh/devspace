import React, { ReactNode } from 'react';
import styles from './HighlightedArea.module.scss';
import { formatError } from 'lib/utils';

interface Props {
  children: ReactNode;
  style?: React.CSSProperties;
  className?: string;
  error?: Error;
}

const HighlightedArea = (props: Props) => {
  return (
    <div
      className={props.className ? styles['highlighted-area'] + ' ' + props.className : styles['highlighted-area']}
      style={props.style}
    >
      <div className={'highlighted-area-content'}>{props.children}</div>
      {props.error && <div className={styles.error}>{formatError(props.error)}</div>}
    </div>
  );
};

export default HighlightedArea;
