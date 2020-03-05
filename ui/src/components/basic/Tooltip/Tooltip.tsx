import React, { ReactNode } from 'react';
import styles from './Tooltip.module.scss';

interface Props {
  position?: 'top' | 'bottom' | 'left' | 'right' | 'for-icon';
  shown?: boolean;
  className?: string;
  text: ReactNode;
  children: ReactNode;
}
interface State {
  isHovered: boolean;
}

class Tooltip extends React.Component<Props, State> {
  state: State = {
    isHovered: false,
  };

  render() {
    const classNames = ['tooltip'];
    const containerClassNames = [styles['tooltip-container'], 'tooltipcontainer'];

    if (this.props.position) {
      switch (this.props.position) {
        case 'top':
          classNames.push(styles['tooltip-top']);
          break;
        case 'bottom':
          classNames.push(styles['tooltip-bottom']);
          break;
        case 'left':
          classNames.push(styles['tooltip-left']);
          break;
        case 'right':
          classNames.push(styles['tooltip-right']);
          break;
        case 'for-icon':
          classNames.push(styles['tooltip-for-icon']);
          containerClassNames.shift();
          containerClassNames.push(styles['tooltip-container-for-icon']);
          break;
        default:
          classNames.push(styles['tooltip-top']);
          return;
      }
    } else {
      classNames.push(styles['tooltip-top']);
    }

    if (!this.props.shown && !this.state.isHovered) {
      classNames.push(styles['tooltip-hidden']);
    } else {
      classNames.push(styles['tooltip-shown']);
    }

    if (this.props.className) {
      containerClassNames.push(this.props.className);
    }

    return (
      <div
        onMouseEnter={() => this.setState({ isHovered: true })}
        onMouseLeave={() => this.setState({ isHovered: false })}
        className={containerClassNames.join(' ')}
      >
        {this.props.children}
        <div className={classNames.join(' ')}>
          <span className={styles['tooltip-arrow']} />
          {this.props.text}
        </div>
      </div>
    );
  }
}

export default Tooltip;
