import React, { ReactNode } from 'react';
import GreenCircle from 'images/green-ellipse.svg';
import RedCircle from 'images/red-ellipse.svg';
import OrangeCircle from 'images/orange-ellipse.svg';
import styles from './StatusIconText.module.scss';
import IconText from 'components/basic/IconText/IconText';

interface Props {
  children: ReactNode;
  status?: string;
}

// CriticalStatus container status
const CriticalStatus = {
  Error: true,
  Unknown: true,
  ImagePullBackOff: true,
  CrashLoopBackOff: true,
  RunContainerError: true,
  ErrImagePull: true,
  CreateContainerConfigError: true,
  InvalidImageName: true,
};

// OkayStatus container status
const OkayStatus = {
  Completed: true,
  Running: true,
};

const StatusIconText = (props: Props) => {
  let icon = OrangeCircle;

  if (CriticalStatus[props.status]) icon = RedCircle;
  if (OkayStatus[props.status]) icon = GreenCircle;

  return (
    <IconText
      tooltip={props.status}
      className={props.status ? styles['status-icon-text'] + ' ' + props.status : styles['status-icon-text']}
      icon={icon}
    >
      {props.children}
    </IconText>
  );
};

export default StatusIconText;
