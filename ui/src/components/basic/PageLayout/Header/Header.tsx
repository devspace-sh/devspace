import React from 'react';
import styles from './Header.module.scss';
import Breadcrumb from 'components/basic/PageLayout/Header/Breadcrumb/Breadcrumb';
import UserMenu from './UserMenu/UserMenu';

const Header = () => {
  return (
    <div className={styles['header-container']}>
      <Breadcrumb />
      <UserMenu />
    </div>
  );
};

export default Header;
