import React from 'react';
import styles from './Header.module.scss';
import Breadcrumb from 'components/basic/PageLayout/Header/Breadcrumb/Breadcrumb';

const Header = () => {
  return (
    <div className={styles['header-container']}>
      <Breadcrumb />
    </div>
  );
};

export default Header;
