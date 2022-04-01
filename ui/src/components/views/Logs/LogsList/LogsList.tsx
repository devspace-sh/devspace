import React from 'react';
import { V1PodList, V1ServiceList, V1Pod } from '@kubernetes/client-node';
import Pod from '../Pod/Pod';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import styles from './LogsList.module.scss';
import inputStyles from '../../../basic/Input/Input.module.scss';
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

const renderPods = (props: Props, searchString: string) => {
  if (searchString !== "") {
    const podMatches = (pod : V1Pod) => (pod.metadata.name.includes(searchString) || pod.spec.containers.some(container => container.name.includes(searchString)));
    props.podList.items = props.podList.items.filter(podMatches);
  }

  if (props.podList.items.length === 0) {
    return <div className={styles['nothing-found']}>No pods found in namespace {props.devSpaceConfig.kubeNamespace}</div>;
  }

  return props.podList.items.map((pod) => {
    const labels = pod.metadata && pod.metadata.labels ? pod.metadata.labels : {};
    let servicePort = '';

    // Check if there is a service that listens to that pod
    if (props.serviceList && props.serviceList.items) {
      for (let i = 0; i < props.serviceList.items.length; i++) {
        const service = props.serviceList.items[i];
        if (service.spec.type === 'ClusterIP' && service.spec.ports && service.spec.ports.length === 1) {
          if (service.spec.ports[0].targetPort && typeof service.spec.ports[0].targetPort === 'string') {
            continue;
          }

          let notFound = false;
          if (service.spec.selector) {
            Object.keys(service.spec.selector).forEach((key) => {
              if (labels[key] !== service.spec.selector[key]) {
                notFound = true;
              }
            });

            if (notFound === false) {
              servicePort =
                service.metadata.name +
                ':' +
                (service.spec.ports[0].targetPort ? (service.spec.ports[0].targetPort as any) : service.spec.ports[0].port);
              break;
            }
          }
        }
      }
    }

    return (
      <Pod
        key={pod.metadata.uid}
        cache={props.cache}
        pod={pod}
        onSelect={props.onSelect}
        service={servicePort}
        selectedContainer={props.selected && props.selected.pod === pod.metadata.name ? props.selected.container : undefined}
      />
    );
  });
};

const LogsList = (props: Props) => {
  const [searchString, setSearchString] = React.useState("");
  return <div className={styles['logs-list']}>
    <div className={styles['logs-list-wrapper']}>
      <form className={styles['search']}>
        <input className={inputStyles['input-component']}
          placeholder="Search pods"
          value={searchString}
          onChange={event => setSearchString(event.target.value)}
        />
      </form>
      {renderPods(props, searchString)}
    </div>
  </div>
};

export default withDevSpaceConfig(LogsList);
