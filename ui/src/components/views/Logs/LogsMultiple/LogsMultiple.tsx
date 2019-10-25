import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { SelectedLogs } from '../LogsList/LogsList';
import styles from './LogsMultiple.module.scss';
import { getDeployedImageNames } from 'lib/utils';
import { Portlet } from 'components/basic/Portlet/Portlet';

interface Props extends DevSpaceConfigContext {
  selected: SelectedLogs;
  onSelect: (selected: SelectedLogs) => void;
}

const LogsMultiple = (props: Props) => (
  <Portlet
    className={
      props.selected && typeof props.selected.multiple === 'object'
        ? styles['logs-multiple'] + ' ' + styles.selected
        : styles['logs-multiple']
    }
    onClick={() =>
      props.onSelect({
        multiple: getDeployedImageNames(props.devSpaceConfig),
      })
    }
  >
    All deployed containers (merged logs)
  </Portlet>
);

export default withDevSpaceConfig(LogsMultiple);
