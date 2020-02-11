import React from 'react';
import EditIcon from 'images/IconButton/settings.svg';
import IconButton, { IconButtonProps } from 'components/basic/IconButton/IconButton';

const EditButton = (props: IconButtonProps) => {
  return (
    <IconButton
      {...props}
      tooltipText={props.tooltipText || 'Edit'}
      className={props.className ? 'edit-button filter ' + props.className : 'edit-button filter'}
      icon={props.icon || EditIcon}
    />
  );
};

export default EditButton;
