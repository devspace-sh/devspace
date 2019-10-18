import React, { ReactNode } from 'react';

interface Props {
  className?: string;
  title?: ReactNode;

  disabled?: boolean;
  children?: ReactNode;
}

const TabContent = (props: Props) => {
  return <div className={props.className ? props.className : ''}>{props.children}</div>;
};

TabContent.role = 'TabContent';

export default TabContent;
