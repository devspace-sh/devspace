import React, { ReactNode } from 'react';
import styles from './Portlet.module.scss';

interface Props {
  className?: string;
  children: ReactNode;
}

export const Portlet = (props: Props) => {
  return <div className={props.className ? styles.portlet + ' ' + props.className : styles.portlet}>{props.children}</div>;
};
