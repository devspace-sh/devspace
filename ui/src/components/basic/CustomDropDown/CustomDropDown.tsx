import React, { RefObject } from 'react';
import styles from './CustomDropDown.module.scss';

export interface DropDownSelectedOption {
  id: string | number;
  text: string;
  markup?: string | Element | JSX.Element;
  disabled?: boolean;
  data?: any; // Leave any here
}

interface Props {
  onChange: (selected: DropDownSelectedOption) => void;
  options: DropDownSelectedOption[];
  className?: string;

  placeholder?: string;
  selectedValue: DropDownSelectedOption;
}

interface State {
  isOpened: boolean;
  prevSelected: string | number;
}

export default class CustomDropDown extends React.PureComponent<Props, State> {
  state: State = {
    isOpened: false,
    prevSelected: null,
  };

  wrapperRef: RefObject<HTMLDivElement> = React.createRef();
  listItemsRef: RefObject<HTMLDivElement> = React.createRef();

  componentDidMount = () => {
    document.addEventListener('mousedown', this.handleClickOutside);
  };

  componentWillUnmount = () => {
    document.removeEventListener('mousedown', this.handleClickOutside);
  };

  handleClickOutside = (event: Event) => {
    if (this.wrapperRef.current && !this.wrapperRef.current.contains(event.target as Node) && this.state.isOpened) {
      this.setState({ isOpened: false });
    }
  };

  onOpen = () => {
    this.setState({ isOpened: !this.state.isOpened });
  };

  onListItemClick = (option: DropDownSelectedOption) => {
    this.props.onChange(option);
    this.setState({ isOpened: false });
  };

  render() {
    const selectedValue = this.props.selectedValue
      ? this.props.selectedValue.text
      : this.props.placeholder
      ? this.props.placeholder
      : 'Select...';
    const classNames = this.props.className
      ? styles['custom-dropdown-wrapper'] + ' ' + this.props.className
      : styles['custom-dropdown-wrapper'];

    const newProps = [];

    if (this.state.isOpened) {
      const elem = this.listItemsRef;
      const bounding = elem.current.getBoundingClientRect();
      const overlapRight = bounding.right > (window.innerWidth || document.documentElement.clientWidth);

      if (overlapRight) newProps.push('push-left');
    }

    return (
      <div
        ref={this.wrapperRef}
        onClick={() => this.onOpen()}
        className={this.state.isOpened ? classNames + ' ' + styles.opened : classNames}
      >
        <div className={'CustomDropDown_selected-value'}>{selectedValue}</div>
        <div
          className={newProps.length > 0 ? styles['list-items'] + ' ' + newProps.join(' ') : styles['list-items']}
          ref={this.listItemsRef}
        >
          {this.props.options.map((option, idx) => {
            return (
              <div
                key={idx}
                className={option.disabled ? styles['option'] + ' ' + styles.disabled : styles['option']}
                onClick={() => (option.disabled ? null : this.onListItemClick(option))}
                title={option.text}
              >
                {option.markup || option.text}
              </div>
            );
          })}
        </div>
      </div>
    );
  }
}
