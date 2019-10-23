import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import style from './Pod.module.scss';
import StatusIconText from 'components/basic/IconText/StatusIconText/StatusIconText';
import { GetPodStatus } from 'lib/utils';
import { SelectedLogs } from '../LogsList/LogsList';
import { Portlet } from 'components/basic/Portlet/Portlet';
import IconButton from 'components/basic/IconButton/IconButton';
import TerminalIconWhite from 'images/icon-terminal-white.svg';
import TerminalIcon from 'images/icon-terminal.svg';

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
          <IconButton
            icon={props.selectedContainer === container.name ? TerminalIconWhite : TerminalIcon}
            tooltipText={'Open terminal'}
            onClick={(e) => {
              e.stopPropagation();
              props.onSelect({
                pod: props.pod.metadata.name,
                container: container.name,
                interactive: true,
              });
            }}
          />
        </div>
      ))}
    </div>
  );
};

const Pod = (props: Props) => {
  const singleContainer = props.pod.spec.containers && props.pod.spec.containers.length === 1;
  const status = GetPodStatus(props.pod);
  const selected = singleContainer && props.selectedContainer;

  return (
    <Portlet
      className={selected ? style.pod + ' ' + style.selected : style.pod}
      onClick={
        singleContainer
          ? () => props.onSelect({ pod: props.pod.metadata.name, container: props.pod.spec.containers[0].name })
          : null
      }
    >
      <StatusIconText status={status as any}>{props.pod.metadata.name}</StatusIconText>
      {!singleContainer && renderContainers(props)}
      {singleContainer && (
        <IconButton
          icon={selected ? TerminalIconWhite : TerminalIcon}
          tooltipText={'Open terminal'}
          onClick={(e) => {
            e.stopPropagation();
            props.onSelect({
              pod: props.pod.metadata.name,
              container: props.pod.spec.containers[0].name,
              interactive: true,
            });
          }}
        />
      )}
    </Portlet>
  );
};

export default Pod;
