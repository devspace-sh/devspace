import React from 'react';
import { V1Pod } from '@kubernetes/client-node';
import styles from './Pod.module.scss';
import StatusIconText from 'components/basic/IconText/StatusIconText/StatusIconText';
import { GetPodStatus, GetContainerStatus, configToYAML } from 'lib/utils';
import { SelectedLogs } from '../LogsList/LogsList';
import IconButton from 'components/basic/IconButton/IconButton';
import TerminalIconExists from 'images/icon-terminal-exists.svg';
import TerminalIconWhite from 'images/icon-terminal-white.svg';
import TerminalIcon from 'images/icon-terminal.svg';
import WarningIcon from 'components/basic/Icon/WarningIcon/WarningIcon';
import LeftAlignIcon from 'images/left-alignment.svg';
import { TerminalCacheInterface } from '../TerminalCache/TerminalCache';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import withPopup, { PopupContext } from 'contexts/withPopup/withPopup';
import AlertPopupContent from 'components/basic/Popup/AlertPopupContent/AlertPopupContent';
import CodeSnippet from 'components/basic/CodeSnippet/CodeSnippet';
import { PortletSimple } from 'components/basic/Portlet/PortletSimple/PortletSimple';

interface Props extends DevSpaceConfigContext, PopupContext {
  pod: V1Pod;
  cache: TerminalCacheInterface;

  selectedContainer?: string;
  onSelect: (selected: SelectedLogs) => void;
}

const exists = (cache: TerminalCacheInterface, selected: SelectedLogs) => {
  return !!cache.terminals.find(
    (terminal) =>
      terminal.pod === selected.pod &&
      terminal.container === selected.container &&
      terminal.interactive === selected.interactive
  );
};

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
    <div className={styles['container-wrapper']}>
      {props.pod.spec.containers.map((container) => {
        const containerStatus = props.pod.status.containerStatuses.find((status) => status.name === container.name);

        return (
          <div
            key={container.name}
            className={
              props.selectedContainer === container.name ? styles.container + ' ' + styles.selected : styles.container
            }
            onClick={() => props.onSelect({ pod: props.pod.metadata.name, container: container.name })}
          >
            <StatusIconText className={styles.status} status={GetContainerStatus(containerStatus)}>
              {container.name}
              {containerStatus && containerStatus.restartCount > 0 && (
                <WarningIcon className={styles.warning} tooltipText={containerStatus.restartCount + ' restarts'} />
              )}
            </StatusIconText>
            <IconButton
              filter={false}
              icon={
                exists(props.cache, { pod: props.pod.metadata.name, container: container.name, interactive: true })
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
      <CodeSnippet lineNumbers={true} className={styles.codesnippet}>
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
  const classNames = [styles.pod];
  if (selected) {
    classNames.push(styles.selected);
  }
  if (singleContainer) {
    classNames.push(styles['single-container']);
  }

  return (
    <PortletSimple
      className={classNames.join(' ')}
      onClick={
        singleContainer
          ? () => props.onSelect({ pod: props.pod.metadata.name, container: props.pod.spec.containers[0].name })
          : null
      }
    >
      {{
        top: {
          left: (
            <StatusIconText className={styles.status + ' ' + styles['status-padding']} status={status}>
              {props.pod.metadata.name}
              {restarts > 0 && <WarningIcon className={styles.warning} tooltipText={restarts + ' restarts'} />}
            </StatusIconText>
          ),
          right: (
            <div className={styles.buttons}>
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
                    exists(props.cache, {
                      pod: props.pod.metadata.name,
                      container: props.pod.spec.containers[0].name,
                      interactive: true,
                    })
                      ? TerminalIconExists
                      : selected
                      ? TerminalIconWhite
                      : TerminalIcon
                  }
                  tooltipText={'Terminal'}
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
          ),
        },
        content: !singleContainer && renderContainers(props),
      }}
    </PortletSimple>
  );
};

export default withDevSpaceConfig(withPopup(Pod));
