import React, { ReactNode } from 'react';
import styles from './PortletSimple.module.scss';
import { Portlet } from 'components/basic/Portlet/Portlet';

interface Props {
  className?: string;
  smallPadding?: boolean;
  onClick?: () => void;

  children: {
    top?: {
      left?: ReactNode;
      right?: ReactNode;
    };

    content?: ReactNode;
  };
}

const renderTop = (props: Props) => {
  if (!props.children.top || (!props.children.top.left && !props.children.top.right)) {
    return null;
  }

  return (
    <div className="top">
      {props.children.top.left && <div className={'left'}>{props.children.top.left}</div>}
      {props.children.top.right && <div className={'right'}>{props.children.top.right}</div>}
    </div>
  );
};

const renderContent = (props: Props) => {
  if (!props.children.content) {
    return null;
  }

  return <div className={'content'}>{props.children.content}</div>;
};

export const PortletSimple = (props: Props) => {
  const classNames = [styles['portlet-simple']];
  if (props.children.content) {
    classNames.push(styles['portlet-with-content']);
  }
  if (props.className) {
    classNames.push(props.className);
  }
  if (props.smallPadding) {
    classNames.push(styles['portlet-padding-small']);
  }

  return (
    <Portlet onClick={props.onClick} className={classNames.join(' ')}>
      {renderTop(props)}
      {renderContent(props)}
    </Portlet>
  );
};
