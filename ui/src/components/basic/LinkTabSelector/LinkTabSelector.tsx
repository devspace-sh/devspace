import React, { ReactNode } from 'react';
import styles from './LinkTabSelector.module.scss';

interface Props {
  className?: string;
  children: ReactNode;
}

interface State {
  isMenuOpen: boolean;
}

export default class LinkTabSelector extends React.PureComponent<Props, State> {
  state: State = {
    isMenuOpen: false,
  };

  menuClick = () => {
    this.setState({ isMenuOpen: !this.state.isMenuOpen });
  };

  render() {
    const classnames = [styles['link-tab-selector']];

    if (this.props.className) classnames.push(this.props.className);
    if (this.state.isMenuOpen) classnames.push(styles['menu-open']);

    return (
      <div className={classnames.join(' ')}>
        <div onClick={() => this.menuClick()}>
          <div className={styles['items']}>{this.props.children}</div>
        </div>
      </div>
    );
  }
}
