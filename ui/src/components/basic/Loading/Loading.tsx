import * as React from 'react';
import styles from './Loading.module.scss';

interface Props {
  className?: string;
}

export default function(props: Props) {
  return (
    <div className={props.className ? styles['loading-wrapper'] + ' ' + props.className : styles['loading-wrapper']}>
      <div className={'loading'}>
        <div className={styles['loading-circle']} />
        <div className={styles['loading-text']}>Loading ...</div>
      </div>
    </div>
  );
}
