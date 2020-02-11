import React from 'react';
import IconCopy from 'images/icon-copy.svg';
import styles from './CopyButton.module.scss';
import { copyToClipboard } from 'lib/utils';
import { Props } from 'react-responsive-select';
import IconButton from 'components/basic/IconButton/IconButton';

interface Props {
  textToCopy: string;
  tooltipText?: string;
  className?: string;
}

export default function CopyButton(props: Props) {
  return (
    <IconButton
      filter={false}
      className={props.className ? styles['copy-button '] + ' ' + props.className : styles['copy-button']}
      icon={IconCopy}
      tooltipText={props.tooltipText}
      onClick={() => copyToClipboard(props.textToCopy)}
    />
  );
}
