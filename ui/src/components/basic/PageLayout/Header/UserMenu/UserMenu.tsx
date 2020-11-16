import React from 'react';
import styles from './UserMenu.module.scss';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import { withRouter, RouteComponentProps } from 'react-router-dom';
import ErrorBoundary from 'components/basic/ErrorBoundary/ErrorBoundary';
import Tooltip from 'components/basic/Tooltip/Tooltip';
import GitHubButton from 'components/basic/GitHubButton/GitHubButton';

interface Props extends PopupContext {}
interface State {
  menuOpen: boolean;
}

class UserMenu extends React.PureComponent<Props & RouteComponentProps, State> {
  state: State = {
    menuOpen: false,
  };

  render() {
    return (
      <ErrorBoundary>
        <div className={this.state.menuOpen ? styles['user-menu'] + ' ' + styles['menu-open'] : styles['user-menu']}>
          <Tooltip className={styles.tooltipcontainer} text="Documentation" position="bottom">
            <a href="https://devspace.sh/docs" target="_blank" className={styles.link + ' ' + styles.docs} />
          </Tooltip>
        </div>
        <GitHubButton />
      </ErrorBoundary>
    );
  }
}

export default withRouter(withPopup(UserMenu));
