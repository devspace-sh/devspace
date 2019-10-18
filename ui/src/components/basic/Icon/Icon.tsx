import React, { ReactNode } from 'react';
import styles from './Icon.module.scss';
import Tooltip from 'components/basic/Tooltip/Tooltip';

interface Props {
  icon: string;
  tooltipText?: ReactNode;
}

const Icon = (props: Props) => {
  if (props.tooltipText) {
    return (
      <Tooltip position="top" text={props.tooltipText}>
        <img className={styles['icon-component']} src={props.icon} />
      </Tooltip>
    );
  } else {
    return <img className={styles['icon-component'] + ' iconcomponent'} src={props.icon} />;
  }
};

export default Icon;
