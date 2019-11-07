import React from 'react';
import styles from './Warning.module.scss';
import CloseButton from 'components/basic/IconButton/CloseButton/CloseButton';

export interface WarningProps {
  uuid: string;
  show?: boolean;

  children: React.ReactNode;
  close: () => void;
}

const Warning = (props: WarningProps) => (
  <div className={styles['warning']} style={{ display: props.show ? 'block' : 'none' }}>
    <div className={styles['wrapper']}>
      <div>{props.children}</div>
      {props.close && (
        <div>
          <CloseButton className={styles['close']} filter={false} white={true} onClick={props.close} />
        </div>
      )}
    </div>
  </div>
);

export default Warning;
