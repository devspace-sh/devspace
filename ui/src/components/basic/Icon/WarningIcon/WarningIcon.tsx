import React, { ReactNode } from 'react';
import Warning from 'images/danger-symbol.svg';
import Icon from 'components/basic/Icon/Icon';

interface Props {
  tooltipText?: ReactNode;
  className?: string;
}

const WarningIcon = (props: Props) => {
  return <Icon icon={Warning} tooltipText={props.tooltipText} className={props.className} />;
};

export default WarningIcon;
