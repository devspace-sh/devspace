import React, { ReactNode } from 'react';
import styles from './AdvancedCodeLine.module.scss';

const AdvancedCodeLine = ({ children }: { children: ReactNode | string }) => {
  return <div className={styles['advanced-code-line']}>{children}</div>;
};

export default AdvancedCodeLine;
