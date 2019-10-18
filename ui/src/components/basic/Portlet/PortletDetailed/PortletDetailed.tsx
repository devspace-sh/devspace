import React, { ReactNode } from 'react';
import styles from './PortletDetailed.module.scss';
import { Portlet } from 'components/basic/Portlet/Portlet';

interface Props {
  className?: string;
  children: {
    top?: {
      left?: ReactNode;
      right?: ReactNode;
    };
    bottom?: {
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
    <div className={'top'}>
      {props.children.top.left && <div className={'left'}>{props.children.top.left}</div>}
      {props.children.top.right && <div className={'right'}>{props.children.top.right}</div>}
    </div>
  );
};

const renderBottom = (props: Props) => {
  if (!props.children.bottom || (!props.children.bottom.left && !props.children.bottom.right)) {
    return null;
  }

  return (
    <div className={'bottom'}>
      {props.children.bottom.left && <div className={'left'}>{props.children.bottom.left}</div>}
      {props.children.bottom.right && <div className={'right'}>{props.children.bottom.right}</div>}
    </div>
  );
};

const renderContent = (props: Props) => {
  if (!props.children.content) {
    return null;
  }

  return <div className={'content'}>{props.children.content}</div>;
};

export const PortletDetailed = (props: Props) => {
  return (
    <Portlet className={props.className ? styles['portlet-detailed'] + ' ' + props.className : styles['portlet-detailed']}>
      {renderTop(props)}
      {renderBottom(props)}
      {renderContent(props)}
    </Portlet>
  );
};
