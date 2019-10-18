import React, { ReactNode } from 'react';
import Warning from 'images/IconButton/danger-symbol.svg';
import Icon from 'components/basic/Icon/Icon';

interface Props {
  tooltipText?: ReactNode;
}

const WarningIcon = (props: Props) => {
  return <Icon icon={Warning} tooltipText={props.tooltipText} />;
};

export default WarningIcon;
