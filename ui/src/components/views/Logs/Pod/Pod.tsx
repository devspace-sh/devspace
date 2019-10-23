import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import style from './Pod.module.scss';
import StatusIconText from 'components/basic/IconText/StatusIconText/StatusIconText';
import { GetPodStatus } from 'lib/utils';

interface Props {
  pod: V1Pod;

  selectedContainer?: string;
  onClickContainer: (container: string) => void;
}

const renderContainers = (props: Props) => {
  return (
    <div>
      {props.pod.spec.containers.map((container) => (
        <div
          className={props.selectedContainer === container.name ? style.container + ' ' + style.selected : style.container}
          onClick={() => props.onClickContainer(container.name)}
        >
          {container.name}
        </div>
      ))}
    </div>
  );
};

const Pod = (props: Props) => {
  const singleContainer = props.pod.spec.containers && props.pod.spec.containers.length === 1;
  const status = GetPodStatus(props.pod);
  console.log(status);

  return (
    <div
      className={singleContainer && props.selectedContainer ? style.pod + ' ' + style.selected : style.pod}
      onClick={singleContainer ? () => props.onClickContainer(props.pod.spec.containers[0].name) : null}
    >
      <StatusIconText status={status as any}>{props.pod.metadata.name}</StatusIconText>
      {!singleContainer && renderContainers(props)}
    </div>
  );
};

export default Pod;
