import React from 'react';
import styles from './ChangeNamespace.module.scss';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';
import CustomDropDown, { DropDownSelectedOption } from 'components/basic/CustomDropDown/CustomDropDown';

interface Props extends DevSpaceConfigContext {}

interface State {
  namespaceValue: DropDownSelectedOption;
  kubecontextValue: DropDownSelectedOption;
}

class ChangeNamespace extends React.PureComponent<Props, State> {
  state: State = {
    namespaceValue: {
      id: this.props.devSpaceConfig.kubeNamespace,
      text: this.props.devSpaceConfig.kubeNamespace,
    },
    kubecontextValue: {
      id: this.props.devSpaceConfig.kubeContext,
      text: this.props.devSpaceConfig.kubeContext,
    },
  };

  render() {
    const classnames = [styles['change-namespace']];

    const namespaceOptions: DropDownSelectedOption[] = [
      {
        id: this.props.devSpaceConfig.kubeNamespace,
        text: this.props.devSpaceConfig.kubeNamespace,
      },
    ];

    const kubeContextOptions: DropDownSelectedOption[] = [
      {
        id: this.props.devSpaceConfig.kubeNamespace,
        text: this.props.devSpaceConfig.kubeNamespace,
      },
    ];

    console.log(this.props.devSpaceConfig);

    return (
      <div className={classnames.join(' ')}>
        <PortletSimple>
          {{
            top: {
              left: (
                <label>
                  Namespace
                  <CustomDropDown
                    className={styles.dropdown}
                    options={namespaceOptions}
                    selectedValue={this.state.namespaceValue}
                    onChange={(selected: DropDownSelectedOption) => {
                      this.setState({ namespaceValue: selected });
                    }}
                  />
                </label>
              ),
              right: (
                <label>
                  KubeContext
                  <CustomDropDown
                    className={styles.dropdown}
                    options={kubeContextOptions}
                    selectedValue={this.state.kubecontextValue}
                    onChange={(selected: DropDownSelectedOption) => {
                      this.setState({ kubecontextValue: selected });
                    }}
                  />
                </label>
              ),
            },
          }}
        </PortletSimple>
        {/* Namespace: */}
        {/* <Input
          placeholder="Namespace"
          value={this.state.namespaceValue}
          onChange={(e) => this.setState({ namespaceValue: e.target.value })}
        />
        {this.props.devSpaceConfig.kubeNamespace !== this.state.namespaceValue && (
          <Button onClick={() => this.props.devSpaceConfig.changeNamespace(this.state.namespaceValue)}>Change</Button>
        )} */}
      </div>
    );
  }
}

export default withDevSpaceConfig(ChangeNamespace);
