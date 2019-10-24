import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import style from './Pod.module.scss';
import StatusIconText from 'components/basic/IconText/StatusIconText/StatusIconText';
import { GetPodStatus, GetContainerStatus, configToYAML } from 'lib/utils';
import { SelectedLogs } from '../LogsList/LogsList';
import { Portlet } from 'components/basic/Portlet/Portlet';
import IconButton from 'components/basic/IconButton/IconButton';
import TerminalIconExists from 'images/icon-terminal-exists.svg';
import TerminalIconWhite from 'images/icon-terminal-white.svg';
import TerminalIcon from 'images/icon-terminal.svg';
import WarningIcon from 'components/basic/Icon/WarningIcon/WarningIcon';
import LeftAlignIcon from 'images/left-alignment.svg';
import TerminalCache from '../TerminalCache/TerminalCache';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import AlertPopupContent from 'components/basic/Popup/AlertPopupContent/AlertPopupContent';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';

interface Props extends DevSpaceConfigContext, PopupContext {
  pod: V1Pod;
  cache: TerminalCache;

  selectedContainer?: string;
  onSelect: (selected: SelectedLogs) => void;
}

const getRestarts = (pod: V1Pod) => {
  let restarts = 0;

  if (pod.status && pod.status.containerStatuses) {
    pod.status.containerStatuses.forEach((status) => {
      if (status.restartCount > restarts) {
        restarts = status.restartCount;
      }
    });
  }

  return restarts;
};

const renderContainers = (props: Props) => {
  return (
    <div className={style['container-wrapper']}>
      {props.pod.spec.containers.map((container) => {
        const containerStatus = props.pod.status.containerStatuses.find((status) => status.name === container.name);

        return (
          <div
            key={container.name}
            className={props.selectedContainer === container.name ? style.container + ' ' + style.selected : style.container}
            onClick={() => props.onSelect({ pod: props.pod.metadata.name, container: container.name })}
          >
            <StatusIconText className={style.status} status={GetContainerStatus(containerStatus)}>
              {container.name}
              {containerStatus && containerStatus.restartCount > 0 && (
                <WarningIcon className={style.warning} tooltipText={containerStatus.restartCount + ' restarts'} />
              )}
            </StatusIconText>
            <IconButton
              filter={false}
              icon={
                props.cache.exists({ pod: props.pod.metadata.name, container: container.name, interactive: true })
                  ? TerminalIconExists
                  : props.selectedContainer === container.name
                  ? TerminalIconWhite
                  : TerminalIcon
              }
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
        );
      })}
    </div>
  );
};

const openYAMLPopup = (props: Props) => {
  props.popup.openPopup(
    <AlertPopupContent hideCloseButton={true} width="1000px" title={'Pod ' + props.pod.metadata.name + ' YAML'}>
      <CodeSnippet lineNumbers={true} className={style.codesnippet}>
        {configToYAML({ apiVersion: 'v1', kind: 'Pod', ...props.pod }, false)}
      </CodeSnippet>
    </AlertPopupContent>
  );
};

const Pod = (props: Props) => {
  const singleContainer = props.pod.spec.containers && props.pod.spec.containers.length === 1;
  const status = GetPodStatus(props.pod);
  const restarts = getRestarts(props.pod);
  const selected = singleContainer && props.selectedContainer;
  const classNames = [style.pod];
  if (selected) {
    classNames.push(style.selected);
  }
  if (singleContainer) {
    classNames.push(style['single-container']);
  }

  return (
    <Portlet
      className={classNames.join(' ')}
      onClick={
        singleContainer
          ? () => props.onSelect({ pod: props.pod.metadata.name, container: props.pod.spec.containers[0].name })
          : null
      }
    >
      <StatusIconText className={style.status + ' ' + style['status-padding']} status={status}>
        {props.pod.metadata.name}
        {restarts > 0 && <WarningIcon className={style.warning} tooltipText={restarts + ' restarts'} />}
      </StatusIconText>
      {!singleContainer && renderContainers(props)}
      <div className={style.buttons}>
        <IconButton
          filter={false}
          icon={LeftAlignIcon}
          tooltipText="Show YAML"
          onClick={(e) => {
            e.stopPropagation();
            openYAMLPopup(props);
          }}
        />
        {singleContainer && (
          <IconButton
            filter={false}
            icon={
              singleContainer &&
              props.cache.exists({
                pod: props.pod.metadata.name,
                container: props.pod.spec.containers[0].name,
                interactive: true,
              })
                ? TerminalIconExists
                : selected
                ? TerminalIconWhite
                : TerminalIcon
            }
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
      </div>
    </Portlet>
  );
};

export default withDevSpaceConfig(withPopup(Pod));
