import React from 'react';
import styles from './ChangeKubeContext.module.scss';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import CustomDropDown, { DropDownSelectedOption } from 'components/basic/CustomDropDown/CustomDropDown';
import { ApiHostname } from 'lib/rest';
import { V1NamespaceList, V1Namespace } from '@kubernetes/client-node';

interface Props extends DevSpaceConfigContext {}

interface State {
  namespaceValue: DropDownSelectedOption;
  kubecontextValue: DropDownSelectedOption;
  namespaceList: V1Namespace[];
}

class ChangeNamespace extends React.PureComponent<Props, State> {
  static getDerivedStateFromProps(nextProps: Props, prevState: State) {
    if (nextProps.devSpaceConfig.kubeContext !== prevState.kubecontextValue.text) {
      return {
        kubecontextValue: {
          id: nextProps.devSpaceConfig.kubeContext,
          text: nextProps.devSpaceConfig.kubeContext,
        },
        namespaceValue: {
          id: nextProps.devSpaceConfig.kubeNamespace,
          text: nextProps.devSpaceConfig.kubeNamespace,
        },
      };
    }
    if (nextProps.devSpaceConfig.kubeNamespace !== prevState.namespaceValue.text) {
      return {
        namespaceValue: {
          id: nextProps.devSpaceConfig.kubeNamespace,
          text: nextProps.devSpaceConfig.kubeNamespace,
        },
      };
    } else return null;
  }

  state: State = {
    namespaceValue: {
      id: this.props.devSpaceConfig.kubeNamespace,
      text: this.props.devSpaceConfig.kubeNamespace,
    },
    kubecontextValue: {
      id: this.props.devSpaceConfig.kubeContext,
      text: this.props.devSpaceConfig.kubeContext,
    },
    namespaceList: null,
  };

  componentDidMount = async () => {
    this.getNamespaces();
  };

  componentDidUpdate = (prevProps: Props) => {
    if (this.props.devSpaceConfig.kubeContext !== prevProps.devSpaceConfig.kubeContext) {
      this.getNamespaces();
    }
  };

  getNamespaces = async () => {
    try {
      const response = await fetch(
        `http://${ApiHostname()}/api/resource?resource=namespaces&context=${this.props.devSpaceConfig.kubeContext}`
      );
      if (response.status !== 200) {
        throw new Error(await response.text());
      }

      const namespaces: V1NamespaceList = await response.json();
      this.setState({ namespaceList: namespaces.items });
    } catch (err) {
      this.setState({ namespaceList: null });
    }
  };

  render() {
    const classnames = [styles['change-namespace']];

    let namespaceOptions: DropDownSelectedOption[] = [
      {
        id: this.props.devSpaceConfig.kubeNamespace,
        text: this.props.devSpaceConfig.kubeNamespace,
      },
    ];

    if (this.state.namespaceList) {
      namespaceOptions = this.state.namespaceList.map((namespace) => {
        return {
          id: namespace.metadata.name,
          text: namespace.metadata.name,
        };
      });
    }

    const kubeContextOptions: DropDownSelectedOption[] = Object.entries(this.props.devSpaceConfig.kubeContexts).map(
      ([key, value]) => {
        return {
          id: key,
          text: key,
          data: {
            namespace: value,
          },
        };
      }
    );

    return (
      <div className={classnames.join(' ')}>
        <PortletSimple>
          {{
            top: {
              left: (
                <label>
                  KubeContext
                  <CustomDropDown
                    className={styles.dropdown}
                    options={kubeContextOptions}
                    selectedValue={this.state.kubecontextValue}
                    onChange={(selected: DropDownSelectedOption) => {
                      this.props.devSpaceConfig.changeKubeContext({
                        contextName: selected.text,
                        contextNamespace: selected.data.namespace,
                      });
                    }}
                  />
                </label>
              ),
              right: (
                <label>
                  Namespace
                  <CustomDropDown
                    className={styles.dropdown}
                    options={namespaceOptions}
                    selectedValue={this.state.namespaceValue}
                    onChange={(selected: DropDownSelectedOption) => {
                      this.props.devSpaceConfig.changeKubeContext({
                        contextNamespace: selected.text,
                      });
                    }}
                  />
                </label>
              ),
            },
          }}
        </PortletSimple>
      </div>
    );
  }
}

export default withDevSpaceConfig(ChangeNamespace);
