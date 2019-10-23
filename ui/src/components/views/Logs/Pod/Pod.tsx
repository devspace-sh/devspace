import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import style from './Pod.module.scss';
import Button from 'components/basic/Button/Button';
import { SelectedLogs } from '../LogsList/LogsList';

interface Props {
  pod: V1Pod;

  selectedContainer?: string;
  onSelect: (selected: SelectedLogs) => void;
}

const renderContainers = (props: Props) => {
  return (
    <div>
      {props.pod.spec.containers.map((container) => (
        <div
          className={props.selectedContainer === container.name ? style.container + ' ' + style.selected : style.container}
          onClick={() => props.onSelect({ pod: props.pod.metadata.name, container: container.name })}
        >
          {container.name}
          <Button
            onClick={(e) => {
              e.stopPropagation();
              props.onSelect({
                pod: props.pod.metadata.name,
                container: container.name,
                interactive: true,
              });
            }}
          >
            enter
          </Button>
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
      onClick={
        singleContainer
          ? () => props.onSelect({ pod: props.pod.metadata.name, container: props.pod.spec.containers[0].name })
          : null
      }
    >
      {props.pod.metadata.name}
      {!singleContainer && renderContainers(props)}
      {singleContainer && (
        <Button
          onClick={(e) => {
            e.stopPropagation();
            props.onSelect({
              pod: props.pod.metadata.name,
              container: props.pod.spec.containers[0].name,
              interactive: true,
            });
          }}
        >
          enter
        </Button>
      )}
    </div>
  );
};

export default Pod;
