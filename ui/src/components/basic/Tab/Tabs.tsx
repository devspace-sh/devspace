import { ReactElement, ReactNode } from 'react';

interface Props {
  selectedIndex?: number;
  children?: ReactNode;
}

export const Tabs = (props: Props): ReactElement => {
  let selectedIndex = 0;
  if (props.selectedIndex) {
    selectedIndex = props.selectedIndex;
  }
  if (selectedIndex < 0) {
    return null;
  }

  if (!props.children) {
    return null;
  }
  if (!(props.children instanceof Array)) {
    return props.children as ReactElement;
  }
  if (props.children.length <= selectedIndex) {
    return null;
  }

  return props.children[selectedIndex] as ReactElement;
};
