import React, { ReactNode } from 'react';
import styles from './AdvancedCodeLine.module.scss';

interface Props {
  className?: string;
  children: ReactNode;
}

const AdvancedCodeLine = (props: Props) => {
  return (
    <div className={props.className ? styles['advanced-code-line'] + ' ' + props.className : styles['advanced-code-line']}>
      {props.children}
    </div>
  );
};

export default AdvancedCodeLine;
