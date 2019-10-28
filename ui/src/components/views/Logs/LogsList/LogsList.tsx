import React from 'react';
import { V1PodList, V1ServiceList } from '@kubernetes/client-node';
import Pod from '../Pod/Pod';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import LogsMultiple from '../LogsMultiple/LogsMultiple';
import { getDeployedImageNames } from 'lib/utils';
import styles from './LogsList.module.scss';
import { TerminalCacheInterface } from '../TerminalCache/TerminalCache';

export interface SelectedLogs {
  pod?: string;
  container?: string;
  interactive?: boolean;
  multiple?: string[];
}

interface Props extends DevSpaceConfigContext {
  podList: V1PodList;
  serviceList: V1ServiceList;
  cache: TerminalCacheInterface;
  onSelect: (selected: SelectedLogs) => void;
  selected?: SelectedLogs;
}

const renderPods = (props: Props) => {
  if (props.podList.items.length === 0) {
    return <div className={styles['nothing-found']}>No pods found in namespace {props.devSpaceConfig.kubeNamespace}</div>;
  }

  return props.podList.items.map((pod) => {
    const labels = pod.metadata && pod.metadata.labels ? pod.metadata.labels : {};
    const ports = {};

    // Check if there is a service that listens to that pod
    if (props.serviceList && props.serviceList.items) {
      for (let i = 0; i < props.serviceList.items.length; i++) {
        const service = props.serviceList.items[i];
        if (service.spec.type === 'ClusterIP' && service.spec.ports && service.spec.ports.length === 1) {
          if (service.spec.ports[0].targetPort && typeof service.spec.ports[0].targetPort === 'string') {
            continue;
          }

          let notFound = false;
          Object.keys(service.spec.selector).forEach((key) => {
            if (labels[key] !== service.spec.selector[key]) {
              notFound = true;
            }
          });

          if (notFound === false) {
            ports[
              service.spec.ports[0].targetPort ? (service.spec.ports[0].targetPort as any) : service.spec.ports[0].port
            ] = true;
          }
        }
      }
    }

    const openPorts = Object.keys(ports);
    return (
      <Pod
        key={pod.metadata.uid}
        cache={props.cache}
        pod={pod}
        onSelect={props.onSelect}
        openPort={openPorts.length === 1 ? openPorts[0] : undefined}
        selectedContainer={props.selected && props.selected.pod === pod.metadata.name ? props.selected.container : undefined}
      />
    );
  });
};

const LogsList = (props: Props) => (
  <div className={styles['logs-list']}>
    <div className={styles['logs-list-wrapper']}>
      {getDeployedImageNames(props.devSpaceConfig).length > 0 &&
        props.devSpaceConfig.kubeNamespace === props.devSpaceConfig.originalKubeNamespace &&
        props.devSpaceConfig.kubeContext === props.devSpaceConfig.originalKubeContext && (
          <LogsMultiple selected={props.selected} onSelect={props.onSelect} />
        )}
      {renderPods(props)}
    </div>
  </div>
);

export default withDevSpaceConfig(LogsList);
