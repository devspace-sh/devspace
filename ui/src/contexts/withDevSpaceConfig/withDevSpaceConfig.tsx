import React from 'react';

const reactDevSpaceConfigContext = React.createContext({
  config: null,
  generatedConfig: null,
  kubeNamespace: null,
  kubeContext: null,
});

const DevSpaceConfigConsumer: React.ExoticComponent<React.ConsumerProps<DevSpaceConfig>> =
  reactDevSpaceConfigContext.Consumer;

export interface DevSpaceConfig {
  config: any;
  generatedConfig: any;

  kubeNamespace: string;
  kubeContext: string;
}

export const DevSpaceConfigContextProvider = reactDevSpaceConfigContext.Provider;

export interface DevSpaceConfigContext {
  devSpaceConfig?: DevSpaceConfig;
}

export default function withDevSpaceConfig<P extends DevSpaceConfigContext>(NewApp: React.ComponentType<P>) {
  return class DevSpaceConfigConsumerComponent extends React.PureComponent<P> {
    render() {
      return (
        <DevSpaceConfigConsumer>
          {(devSpaceConfig: DevSpaceConfig) => <NewApp devSpaceConfig={devSpaceConfig} {...this.props} />}
        </DevSpaceConfigConsumer>
      );
    }
  };
}
