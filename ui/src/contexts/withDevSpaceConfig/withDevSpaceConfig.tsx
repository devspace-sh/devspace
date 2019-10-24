import React from 'react';

const reactDevSpaceConfigContext = React.createContext({
  changeNamespace: (_: string) => null,
  config: null,
  generatedConfig: null,
  profile: null,
  kubeNamespace: null,
  kubeContext: null,
});

const DevSpaceConfigConsumer: React.ExoticComponent<React.ConsumerProps<DevSpaceConfig>> =
  reactDevSpaceConfigContext.Consumer;

export interface DevSpaceConfig {
  changeNamespace: (newNamespace: string) => void;

  config: Config;
  generatedConfig: GeneratedConfig;

  profile: string;
  kubeNamespace: string;
  kubeContext: string;
}

// TODO: complete
interface Config {
  version: string;

  images: { [key: string]: ImageConfig };
}

interface ImageConfig {
  image: string;
}

interface GeneratedConfig {
  vars: { [key: string]: string };
  profiles: { [key: string]: GeneratedCacheConfig };
}

interface GeneratedCacheConfig {
  deployments: { [key: string]: GeneratedDeploymentCache };
  images: { [key: string]: GeneratedImageCache };
  dependencies: { [key: string]: string };
  lastContext: GeneratedLastContextConfig;
}

interface GeneratedImageCache {
  imageConfigHash: string;
  dockerfileHash: string;
  contextHash: string;
  entrypointHash: string;

  customFilesHash: string;

  imageName: string;
  tag: string;
}

interface GeneratedDeploymentCache {
  deploymentConfigHash: string;

  helmOverridesHash: string;
  helmChartHash: string;
  kubectlManifestsHash: string;
}

interface GeneratedLastContextConfig {
  namespace: string;
  context: string;
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
