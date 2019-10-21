import React from 'react';
import { V1PodList } from '@kubernetes/client-node';
import Pod from '../Pod/Pod';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';

interface Props extends DevSpaceConfigContext {
  podList: V1PodList;
  onSelect: (podName: string, containerName: string) => void;
  selected?: {
    pod: string;
    container: string;
  };
}

const renderPods = (props: Props) => {
  if (props.podList.items.length === 0) {
    return `No pods found in namespace ${props.devSpaceConfig.kubeNamespace}`;
  }

  return props.podList.items.map((pod) => (
    <Pod
      key={pod.metadata.uid}
      pod={pod}
      onClickContainer={(container) => props.onSelect(pod.metadata.name, container)}
      selectedContainer={props.selected && props.selected.pod === pod.metadata.name ? props.selected.container : undefined}
    />
  ));
};

const LogsList = (props: Props) => <div>{renderPods(props)}</div>;

export default withDevSpaceConfig(LogsList);
