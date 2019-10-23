import React from 'react';
import { V1PodList } from '@kubernetes/client-node';
import Pod from '../Pod/Pod';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import LogsMultiple from '../LogsMultiple/LogsMultiple';
import { getDeployedImageNames } from 'lib/utils';
import styles from './LogsList.module.scss';

export interface SelectedLogs {
  pod?: string;
  container?: string;
  interactive?: boolean;
  multiple?: string[];
}

interface Props extends DevSpaceConfigContext {
  podList: V1PodList;
  onSelect: (selected: SelectedLogs) => void;
  selected?: SelectedLogs;
}

const renderPods = (props: Props) => {
  if (props.podList.items.length === 0) {
    return `No pods found in namespace ${props.devSpaceConfig.kubeNamespace}`;
  }

  return props.podList.items.map((pod) => (
    <Pod
      key={pod.metadata.uid}
      pod={pod}
      onSelect={props.onSelect}
      selectedContainer={props.selected && props.selected.pod === pod.metadata.name ? props.selected.container : undefined}
    />
  ));
};

const LogsList = (props: Props) => (
  <div className={styles['logs-list']}>
    {getDeployedImageNames(props.devSpaceConfig).length > 0 && (
      <LogsMultiple selected={props.selected} onSelect={props.onSelect} />
    )}
    {renderPods(props)}
  </div>
);

export default withDevSpaceConfig(LogsList);
