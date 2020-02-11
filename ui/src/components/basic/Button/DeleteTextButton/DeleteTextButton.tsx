import React from 'react';
import IconTrashWhite from 'images/trash-white.svg';
import './DeleteTextButton.scss';
import Button, { ButtonProps } from 'components/basic/Button/Button';

interface Props extends ButtonProps {}

const DeleteTextButton = (props: Props) => {
  const classNames = ['delete-text-button'];
  if (props.className) {
    classNames.push(props.className);
  }

  return <Button {...props} className={classNames.join(' ')} icon={props.icon || IconTrashWhite} />;
};

export default DeleteTextButton;
