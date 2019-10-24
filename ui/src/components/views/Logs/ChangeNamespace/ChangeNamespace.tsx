import React from 'react';
import style from './ChangeNamespace.module.scss';
import Input from 'components/basic/Input/Input';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import Button from 'components/basic/Button/Button';

interface Props extends DevSpaceConfigContext {}

interface State {
  namespaceValue: string;
}

class ChangeNamespace extends React.PureComponent<Props, State> {
  state: State = {
    namespaceValue: this.props.devSpaceConfig.kubeNamespace,
  };

  render() {
    const classnames = [style['change-namespace']];

    return (
      <div className={classnames.join(' ')}>
        Namespace:
        <Input
          placeholder="Namespace"
          value={this.state.namespaceValue}
          onChange={(e) => this.setState({ namespaceValue: e.target.value })}
        />
        {this.props.devSpaceConfig.kubeNamespace !== this.state.namespaceValue && (
          <Button onClick={() => this.props.devSpaceConfig.changeNamespace(this.state.namespaceValue)}>Change</Button>
        )}
      </div>
    );
  }
}

export default withDevSpaceConfig(ChangeNamespace);
