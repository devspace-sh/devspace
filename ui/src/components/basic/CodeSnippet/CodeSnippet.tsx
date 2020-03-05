import * as React from 'react';
import styles from './CodeSnippet.module.scss';
import CopyButton from 'components/basic/IconButton/CopyButton/CopyButton';

interface Props {
  copy?: boolean;
  lineOffset?: number;
  children: any;
  lineNumbers?: boolean;
  className?: string;
}

interface State {
  lines: any[];
}

const getTextFromObject = (obj: any): string => {
  if (obj.props && obj.props.children) {
    const children = obj.props.children;
    if (children instanceof Array) {
      const strings = [];
      for (let i = 0; i < children.length; i++) {
        strings.push(getTextFromObject(children[i]));
      }

      return strings.join('\n');
    }

    return getTextFromObject(children);
  }

  return '' + obj;
};

export default class CodeSnippet extends React.PureComponent<Props, State> {
  state: State = {
    lines: [],
  };

  renderLines() {
    if (this.props.lineNumbers) {
      let lineNumber = this.props.lineOffset || 1;

      this.state.lines =
        this.props.children instanceof Array ? this.props.children.join('').split('\n') : this.props.children.split('\n');

      return (
        <div className={styles['code-text']}>
          {this.state.lines.map((line: any) => {
            const re = (
              <span key={lineNumber} className={styles['line']}>
                <span key={lineNumber} className={styles['line-number']}>
                  {lineNumber}
                </span>{' '}
                <span dangerouslySetInnerHTML={{ __html: line.replace(/\s/g, '&nbsp;').trim() }} />
              </span>
            );
            if (lineNumber >= 0) {
              lineNumber++;
            }
            return re;
          })}
        </div>
      );
    } else {
      return this.props.children;
    }
  }

  render() {
    return (
      <div className={this.props.className ? styles['code-snippet'] + ' ' + this.props.className : styles['code-snippet']}>
        {this.props.copy !== false && <CopyButton textToCopy={getTextFromObject(this)} />}
        <div className={styles['code']}>{this.renderLines()}</div>
      </div>
    );
  }
}
