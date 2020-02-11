import React from 'react';
import MaximizeIcon from 'images/maximize.svg';
import MinimizeIcon from 'images/minimize.svg';
import IconButton, { IconButtonProps } from 'components/basic/IconButton/IconButton';

interface Props extends IconButtonProps {
  maximized: boolean;
}

const MaximizeButton = (props: Props) => {
  return (
    <IconButton
      {...props}
      tooltipText={!props.maximized ? 'Maximize' : 'Minimize'}
      className={props.className ? 'maximize-button ' + props.className : 'maximize-button'}
      icon={!props.maximized ? MaximizeIcon : MinimizeIcon}
    />
  );
};

export default MaximizeButton;
