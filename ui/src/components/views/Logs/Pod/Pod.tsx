import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import style from './Pod.module.scss';

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

  return (
    <div
      className={singleContainer && props.selectedContainer ? style.pod + ' ' + style.selected : style.pod}
      onClick={singleContainer ? () => props.onClickContainer(props.pod.spec.containers[0].name) : null}
    >
      {props.pod.metadata.name}
      {!singleContainer && renderContainers(props)}
    </div>
  );
};

export default Pod;
