import React, { ReactNode } from 'react';
import styles from './BaseLayout.module.scss';
import ErrorBoundary from 'components/basic/ErrorBoundary/ErrorBoundary';

const BaseLayout = (props: { children?: ReactNode }) => {
  return (
    <div className={styles['header-layout']}>
      <ErrorBoundary>
        <div className={styles.body}>{props.children}</div>
      </ErrorBoundary>
    </div>
  );
};

export default BaseLayout;
