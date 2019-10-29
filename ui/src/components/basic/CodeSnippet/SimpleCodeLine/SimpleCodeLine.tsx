import React, { ReactNode } from 'react';
import styles from './SimpleCodeLine.module.scss';

const SimpleCodeLine = ({ children }: { children: ReactNode | string }) => {
  return <div className={styles['simple-code-line']}>{children}</div>;
};

export default SimpleCodeLine;
