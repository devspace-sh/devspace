import React, { ReactNode } from 'react';
import styles from './TabSelector.module.scss';

export interface TabSelectorType {
  onSelectTab: (selectedIndex: number) => void;
}

export interface TabSelectorOption {
  index: number;
  disabled?: boolean;
  value: ReactNode;
}

interface Props extends TabSelectorType {
  selectedIndex?: number;
  className?: string;

  options: TabSelectorOption[];
}

const renderOptions = (props: Props) => {
  return props.options.map((option) => {
    const classNames = [styles['option']];
    if (option.index === props.selectedIndex) {
      classNames.push(styles.selected);
    }
    if (option.disabled) {
      classNames.push(styles.disabled);
    }

    return (
      <div
        key={option.index}
        className={classNames.join(' ')}
        onClick={
          !option.disabled
            ? () => {
                props.onSelectTab(option.index);
              }
            : null
        }
      >
        {option.value}
      </div>
    );
  });
};

export const TabSelector = (props: Props) => {
  return (
    <div className={props.className ? styles['tab-selector'] + ' ' + props.className : styles['tab-selector']}>
      {renderOptions(props)}
    </div>
  );
};
