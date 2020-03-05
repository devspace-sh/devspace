import React, { ReactNode } from 'react';
import styles from './PortletTabs.module.scss';
import { TabSelectorType, TabSelectorOption, TabSelector } from 'components/basic/Tab/TabSelector/TabSelector';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import { Tabs } from 'components/basic/Tab/Tabs';

interface State {
  selectedTabIndex: number;
}

interface Props {
  title: ReactNode;

  className?: string;
  smallPadding?: boolean;
  tabSelector?: React.ComponentType<TabSelectorType>;

  children?: ReactNode;

  // Stateful version
  defaultSelectedTabIndex?: number;
  closable?: boolean;
}

export class PortletTabs extends React.Component<Props, State> {
  state: State = {
    selectedTabIndex: 0,
  };

  constructor(props: Props) {
    super(props);

    if (props.defaultSelectedTabIndex) {
      this.state.selectedTabIndex = props.defaultSelectedTabIndex;
    }
  }

  renderTabSelector = () => {
    if (this.props.tabSelector) {
      return <this.props.tabSelector onSelectTab={(selectedIndex) => this.setState({ selectedTabIndex: selectedIndex })} />;
    }

    // Build options map
    if (!this.props.children || !(this.props.children instanceof Array)) {
      return null;
    }

    // Loop over tab contents
    const options: TabSelectorOption[] = this.props.children.map((child, idx) => {
      if ((child as JSX.Element).type && (child as JSX.Element).type.role === 'TabContent') {
        return {
          index: idx,
          value: (child as JSX.Element).props.title,
          disabled: (child as JSX.Element).props.disabled,
        };
      }

      return {
        index: idx,
        value: 'Option ' + idx,
      };
    });

    if (options.length === 0) {
      return null;
    }

    return (
      <TabSelector
        selectedIndex={this.state.selectedTabIndex}
        options={options}
        onSelectTab={(idx) => {
          if (this.props.closable && this.state.selectedTabIndex === idx) {
            this.setState({ selectedTabIndex: -1 });
          } else {
            this.setState({ selectedTabIndex: idx });
          }
        }}
      />
    );
  };

  render() {
    const showContent =
      this.state.selectedTabIndex >= 0 &&
      (!this.props.children ||
        !(this.props.children instanceof Array) ||
        this.state.selectedTabIndex < this.props.children.length);

    return (
      <PortletSimple
        smallPadding={this.props.smallPadding}
        className={this.props.className ? styles['portlet-tabs'] + ' ' + this.props.className : styles['portlet-tabs']}
      >
        {{
          top: {
            left: <div className={styles['portlet-title']}>{this.props.title}</div>,
            right: <div className={styles['portlet-selector']}>{this.renderTabSelector()}</div>,
          },
          content: showContent && <Tabs selectedIndex={this.state.selectedTabIndex}>{this.props.children}</Tabs>,
        }}
      </PortletSimple>
    );
  }
}
