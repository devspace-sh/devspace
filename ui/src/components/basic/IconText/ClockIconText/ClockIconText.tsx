import React, { ReactNode } from 'react';
import styles from './ClockIconText.module.scss';
import ClockIcon from 'images/clock.svg';
import IconText from 'components/basic/IconText/IconText';

interface Props {
  children: ReactNode;
}

const ClockIconText = (props: Props) => {
  return (
    <IconText className={styles['clock-icon-text']} icon={ClockIcon}>
      {props.children}
    </IconText>
  );
};

export default ClockIconText;
