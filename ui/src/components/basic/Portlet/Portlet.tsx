import React, { ReactNode } from 'react';
import styles from './Portlet.module.scss';

interface Props {
  className?: string;
  onClick?: () => void;
  children: ReactNode;
}

export const Portlet = (props: Props) => {
  return (
    <div onClick={props.onClick} className={props.className ? styles.portlet + ' ' + props.className : styles.portlet}>
      {props.children}
    </div>
  );
};
