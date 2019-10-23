import React from 'react';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { SelectedLogs } from '../LogsList/LogsList';
import style from './LogsMultiple.module.scss';
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
        ? style['logs-multiple'] + ' ' + style.selected
        : style['logs-multiple']
    }
    onClick={() =>
      props.onSelect({
        multiple: getDeployedImageNames(props.devSpaceConfig),
      })
    }
  >
    All deployed containers (Merged Logs)
  </Portlet>
);

export default withDevSpaceConfig(LogsMultiple);
