import React, { ReactNode } from 'react';
import GreenCircle from 'images/green-ellipse.svg';
import DarkGreenCircle from 'images/dark-green-ellipse.svg';
import RedCircle from 'images/red-ellipse.svg';
import OrangeCircle from 'images/orange-ellipse.svg';
import styles from './StatusIconText.module.scss';
import IconText from 'components/basic/IconText/IconText';

interface Props {
  children: ReactNode;
  className?: string;
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
  Running: true,
};

// CompletedStatus container status
const CompletedStatus = {
  Completed: true,
};

const StatusIconText = (props: Props) => {
  let icon = OrangeCircle;

  if (CriticalStatus[props.status]) icon = RedCircle;
  if (OkayStatus[props.status]) icon = GreenCircle;
  if (CompletedStatus[props.status]) icon = DarkGreenCircle
  const classNames = [styles['status-icon-text']];
  if (props.status) {
    classNames.push(props.status);
  }
  if (props.className) {
    classNames.push(props.className);
  }

  return (
    <IconText tooltip={props.status} className={classNames.join(' ')} icon={icon}>
      {props.children}
    </IconText>
  );
};

export default StatusIconText;
