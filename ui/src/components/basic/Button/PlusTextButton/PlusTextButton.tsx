import React from 'react';
import IconPlus from 'images/icon-plus.svg';
import Button, { ButtonProps } from 'components/basic/Button/Button';

interface Props extends ButtonProps {}

const PlusTextButton = (props: Props) => {
  const classNames = ['plus-text-button'];
  if (props.className) {
    classNames.push(props.className);
  }

  return <Button {...props} className={classNames.join(' ')} icon={props.icon || IconPlus} />;
};

export default PlusTextButton;
