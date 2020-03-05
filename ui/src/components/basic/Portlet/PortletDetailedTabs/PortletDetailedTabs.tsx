import React, { ReactNode } from 'react';
import styles from './PortletDetailedTabs.module.scss';
import { TabSelectorType, TabSelector, TabSelectorOption } from 'components/basic/Tab/TabSelector/TabSelector';
import { PortletDetailed } from 'components/basic/Portlet/PortletDetailed/PortletDetailed';
import { Tabs } from 'components/basic/Tab/Tabs';

interface State {
  selectedTabIndex: number;
}

interface Props {
  className?: string;
  children: {
    top: {
      left?: ReactNode;
      right?: ReactNode;
    };
    bottom?: {
      left?: ReactNode;
    };
    content: ReactNode;
  };

  tabSelector?: React.ComponentType<TabSelectorType>;

  // Stateless version
  onTabSelected?: (selectedIndex: number) => void;
  selectedTabIndex?: number;

  // Stateful version
  defaultSelectedTabIndex?: number;
  closable?: boolean;
}

export class PortletDetailedTabs extends React.PureComponent<Props, State> {
  state: State = {
    selectedTabIndex: this.props.defaultSelectedTabIndex ? this.props.defaultSelectedTabIndex : 0,
  };

  renderTabSelector = (selectedIndex: number) => {
    if (this.props.tabSelector) {
      return <this.props.tabSelector onSelectTab={(idx) => this.setState({ selectedTabIndex: idx })} />;
    }

    // Build options map
    if (!this.props.children.content || !(this.props.children.content instanceof Array)) {
      if (
        (this.props.children.content as JSX.Element).type &&
        (this.props.children.content as JSX.Element).type.role === 'TabContent'
      ) {
        return (
          <TabSelector
            options={[
              {
                index: 0,
                value: (this.props.children.content as JSX.Element).props.title,
                disabled: (this.props.children.content as JSX.Element).props.disabled,
              },
            ]}
            selectedIndex={selectedIndex}
            onSelectTab={(idx) => {
              if (typeof this.props.selectedTabIndex !== 'undefined') {
                if (this.props.onTabSelected) {
                  this.props.onTabSelected(idx);
                }
              } else {
                if (this.props.closable && this.state.selectedTabIndex === idx) {
                  this.setState({ selectedTabIndex: -1 });
                } else {
                  this.setState({ selectedTabIndex: idx });
                }
              }
            }}
          />
        );
      }

      return null;
    }

    // Loop over tab contents
    const options: TabSelectorOption[] = this.props.children.content.map((child, idx) => {
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
        options={options}
        selectedIndex={selectedIndex}
        onSelectTab={(idx) => {
          if (typeof this.props.selectedTabIndex !== 'undefined') {
            if (this.props.onTabSelected) {
              this.props.onTabSelected(idx);
            }
          } else {
            if (this.props.closable && this.state.selectedTabIndex === idx) {
              this.setState({ selectedTabIndex: -1 });
            } else {
              this.setState({ selectedTabIndex: idx });
            }
          }
        }}
      />
    );
  };

  render() {
    const selectedIndex =
      typeof this.props.selectedTabIndex !== 'undefined' ? this.props.selectedTabIndex : this.state.selectedTabIndex;
    const showContent =
      selectedIndex >= 0 &&
      (!this.props.children.content ||
        !(this.props.children.content instanceof Array) ||
        selectedIndex < this.props.children.content.length);

    return (
      <PortletDetailed
        className={
          this.props.className
            ? styles['portlet-detailed-tabs'] + ' ' + this.props.className
            : styles['portlet-detailed-tabs']
        }
      >
        {{
          top: this.props.children.top,
          bottom: {
            left: this.props.children.bottom ? this.props.children.bottom.left : null,
            right: this.renderTabSelector(selectedIndex),
          },
          content: showContent && <Tabs selectedIndex={selectedIndex}>{this.props.children.content}</Tabs>,
        }}
      </PortletDetailed>
    );
  }
}
