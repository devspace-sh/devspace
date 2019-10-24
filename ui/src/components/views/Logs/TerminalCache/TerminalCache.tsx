import LogsTerminal, { LogsTerminalProps } from 'components/views/Logs/LogsTerminal/LogsTerminal';
import { V1PodList } from '@kubernetes/client-node';
import React from 'react';
import { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { ApiHostname } from '../../../../lib/rest';
import AdvancedCodeLine from 'components/basic/CodeSnippet/AdvancedCodeLine/AdvancedCodeLine';
import style from './TerminalCache.module.scss';

export interface TerminalCacheInterface {
  multiLog?: {
    multiple: string[];
    props: LogsTerminalProps;
  };
  terminals: Array<{
    pod: string;
    container: string;
    interactive: boolean;
    props: LogsTerminalProps;
  }>;
}

class TerminalCache {
  private closed: boolean = false;
  private namespace: string;
  private onDelete: (selected: SelectedLogs) => void;
  private cache: TerminalCacheInterface = {
    terminals: [],
  };

  constructor(namespace: string, onDelete: (selected: SelectedLogs) => void) {
    this.namespace = namespace;
    this.onDelete = onDelete;
  }

  public exists(selected: SelectedLogs) {
    return !!this.cache.terminals.find(
      (terminal) =>
        terminal.pod === selected.pod &&
        terminal.container === selected.container &&
        terminal.interactive === selected.interactive
    );
  }

  public updateNamespace(namespace: string): boolean {
    if (namespace !== this.namespace) {
      this.namespace = namespace;
      this.cache.terminals = [];
      this.cache.multiLog = null;
      return true;
    }

    return false;
  }

  public updateCache(podList: V1PodList) {
    for (let i = 0; i < this.cache.terminals.length; i++) {
      const selectedPod = podList && podList.items.find((pod) => this.cache.terminals[i].pod === pod.metadata.name);
      if (!selectedPod) {
        this.cache.terminals.splice(i, 1);
        i--;
        continue;
      }

      const selectedContainer = selectedPod.spec.containers.find(
        (container) => this.cache.terminals[i].container === container.name
      );
      if (!selectedContainer) {
        this.cache.terminals.splice(i, 1);
        i--;
        continue;
      }
    }
  }

  public select(selected: SelectedLogs) {
    let found = false;
    for (let i = 0; i < this.cache.terminals.length; i++) {
      this.cache.terminals[i].props.show =
        selected &&
        this.cache.terminals[i].pod === selected.pod &&
        this.cache.terminals[i].container === selected.container &&
        this.cache.terminals[i].interactive === selected.interactive;
      found = found || this.cache.terminals[i].props.show;
    }

    if (this.cache.multiLog) {
      this.cache.multiLog.props.show = false;
    }

    if (selected && selected.multiple) {
      this.cache.multiLog = {
        multiple: selected.multiple,
        props: {
          url: `ws://${ApiHostname()}/api/logs-multiple?namespace=${this.namespace}&imageSelector=${selected.multiple.join(
            '&imageSelector='
          )}`,
          interactive: false,
          show: true,
        },
      };
    } else if (selected && selected.pod && !found) {
      this.cache.terminals.push({
        pod: selected.pod,
        container: selected.container,
        interactive: selected.interactive,
        props: {
          url: `ws://${ApiHostname()}/api/${selected.interactive ? 'enter' : 'logs'}?namespace=${this.namespace}&name=${
            selected.pod
          }&container=${selected.container}`,
          interactive: selected.interactive,
          show: true,
        },
      });
    }
  }

  public delete(selected: SelectedLogs) {
    if (this.closed || !selected) {
      return;
    }

    if (selected.pod) {
      const idx = this.cache.terminals.findIndex(
        (terminal) =>
          terminal.pod === selected.pod &&
          terminal.container === selected.container &&
          terminal.interactive === selected.interactive
      );
      if (idx !== -1) {
        this.cache.terminals.splice(idx, 1);
        this.onDelete(selected);
      }
    } else if (selected.multiple) {
      this.cache.multiLog = null;
      this.onDelete(selected);
    }
  }

  public close() {
    this.closed = true;
  }

  public renderTerminals() {
    const terminals = [];

    if (this.cache.multiLog) {
      terminals.push(
        <LogsTerminal
          key="multi-logs"
          {...this.cache.multiLog.props}
          onClose={() => this.delete({ multiple: this.cache.multiLog.multiple })}
        />
      );
    }

    terminals.push(
      ...this.cache.terminals.map((terminal) => (
        <LogsTerminal
          key={terminal.pod + ':' + terminal.container + ':' + (terminal.interactive ? 'interactive' : 'non-interactive')}
          {...terminal.props}
          firstLine={
            <AdvancedCodeLine className={style['first-line']}>
              devspace {terminal.interactive ? 'enter' : 'logs'} -n {this.namespace} --pod {terminal.pod} -c{' '}
              {terminal.container}
            </AdvancedCodeLine>
          }
          onClose={() =>
            this.delete({ pod: terminal.pod, container: terminal.container, interactive: terminal.interactive })
          }
        />
      ))
    );

    return terminals;
  }
}

export default TerminalCache;
