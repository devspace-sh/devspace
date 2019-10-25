import React from 'react';
import styles from './Menu.module.scss';
import { Link } from 'react-router-dom';
import CustomNavLink from 'components/basic/CustomNavLink/CustomNavLink';

interface Props {}

interface State {
  open: boolean;
}

class Menu extends React.PureComponent<Props, State> {
  state: State = {
    open: false,
  };

  render() {
    return (
      <div className={styles['menu-container-wrapper']}>
        <div
          className={
            this.state.open
              ? styles.hamburger + ' ' + styles['is-active'] + ' ' + styles['hamburger--vortex']
              : styles.hamburger + ' ' + styles['hamburger--vortex']
          }
          onClick={() => this.setState({ open: !this.state.open })}
        >
          <div className={styles['hamburger-box']}>
            <div className={styles['hamburger-inner']} />
          </div>
        </div>
        <div className={this.state.open ? styles.menu + ' ' + styles.open : styles.menu}>
          <div>
            <Link to="/">
              <span className={styles.logo} />
            </Link>
            <nav>
              <ul>
                <li>
                  <CustomNavLink className={styles.logs} to="/logs/containers" activeClassName={styles.selected}>
                    Logs
                  </CustomNavLink>
                </li>
                <li>
                  <CustomNavLink className={styles.stack} to="/stack/configuration" activeClassName={styles.selected}>
                    Stack
                  </CustomNavLink>
                </li>
                <li>
                  <CustomNavLink className={styles.commands} to="/commands/commands" activeClassName={styles.selected}>
                    Commands
                  </CustomNavLink>
                </li>
              </ul>
            </nav>
          </div>
        </div>
      </div>
    );
  }
}

export default Menu;
