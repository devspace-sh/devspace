import React, { ReactNode } from 'react';
import styles from './IconText.module.scss';
import Tooltip from 'components/basic/Tooltip/Tooltip';

interface Props {
  icon: string;
  children: ReactNode;
  className?: string;
  tooltip?: string;
}

const IconText = (props: Props) => {
  return (
    <div className={props.className ? styles['icon-text'] + ' ' + props.className : styles['icon-text']}>
      {props.tooltip ? (
        <Tooltip position="top" text={props.tooltip}>
          <img src={props.icon} />
        </Tooltip>
      ) : (
        <img src={props.icon} />
      )}
      <span className={styles['text']}>{props.children}</span>
    </div>
  );
};

export default IconText;
