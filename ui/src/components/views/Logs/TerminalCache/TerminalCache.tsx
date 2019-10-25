import InteractiveTerminal, { InteractiveTerminalProps } from 'components/advanced/InteractiveTerminal/InteractiveTerminal';
import { V1PodList } from '@kubernetes/client-node';
import React from 'react';
import { SelectedLogs } from 'components/views/Logs/LogsList/LogsList';
import { ApiHostname } from '../../../../lib/rest';
import AdvancedCodeLine from 'components/basic/CodeSnippet/AdvancedCodeLine/AdvancedCodeLine';
import style from './TerminalCache.module.scss';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';

export interface TerminalCacheInterface {
  kubeContext: string;
  kubeNamespace: string;

  multiLog?: {
    multiple: string[];
    props: InteractiveTerminalProps;
  };
  terminals: Array<{
    pod: string;
    container: string;
    interactive: boolean;
    props: InteractiveTerminalProps;
  }>;
}

interface Props extends DevSpaceConfigContext {
  podList: V1PodList;
  selected: SelectedLogs;
  onDelete: (selected: SelectedLogs) => void;

  children: (obj: { terminals: React.ReactNode[]; cache: TerminalCacheInterface }) => React.ReactNode;
}

interface State {}

class TerminalCache extends React.PureComponent<Props, State> {
  private closed: boolean = false;
  private selected: SelectedLogs = this.props.selected;
  private cache: TerminalCacheInterface = {
    kubeNamespace: this.props.devSpaceConfig.kubeNamespace,
    kubeContext: this.props.devSpaceConfig.kubeContext,
    terminals: [],
  };

  select = (selected: SelectedLogs) => {
    this.selected = selected;

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
          url: `ws://${ApiHostname()}/api/logs-multiple?context=${this.cache.kubeContext}&namespace=${
            this.cache.kubeNamespace
          }&imageSelector=${selected.multiple.join('&imageSelector=')}`,
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
          url: `ws://${ApiHostname()}/api/${selected.interactive ? 'enter' : 'logs'}?context=${
            this.cache.kubeContext
          }&namespace=${this.cache.kubeNamespace}&name=${selected.pod}&container=${selected.container}`,
          interactive: selected.interactive,
          show: true,
        },
      });
    }
  };

  delete(selected: SelectedLogs) {
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
        this.props.onDelete(selected);
      }
    } else if (selected.multiple) {
      this.cache.multiLog = null;
      this.props.onDelete(selected);
    }
  }

  componentWillUnmount() {
    this.closed = true;
  }

  componentDidMount() {
    this.update();
  }

  componentDidUpdate() {
    this.update();
  }

  render() {
    const terminals = [];
    if (this.cache.multiLog) {
      terminals.push(
        <InteractiveTerminal
          key="multi-logs"
          {...this.cache.multiLog.props}
          onClose={() => this.delete({ multiple: this.cache.multiLog.multiple })}
        />
      );
    }

    terminals.push(
      ...this.cache.terminals.map((terminal) => (
        <InteractiveTerminal
          key={terminal.pod + ':' + terminal.container + ':' + (terminal.interactive ? 'interactive' : 'non-interactive')}
          {...terminal.props}
          firstLine={
            <AdvancedCodeLine className={style['first-line']}>
              devspace {terminal.interactive ? 'enter' : 'logs'} -n {this.cache.kubeNamespace} --pod {terminal.pod} -c{' '}
              {terminal.container}
            </AdvancedCodeLine>
          }
          onClose={() =>
            this.delete({ pod: terminal.pod, container: terminal.container, interactive: terminal.interactive })
          }
        />
      ))
    );

    return this.props.children({ terminals, cache: this.cache });
  }

  private update() {
    if (
      this.props.devSpaceConfig.kubeNamespace !== this.cache.kubeNamespace ||
      this.props.devSpaceConfig.kubeContext !== this.cache.kubeContext
    ) {
      this.cache.kubeNamespace = this.props.devSpaceConfig.kubeNamespace;
      this.cache.kubeContext = this.props.devSpaceConfig.kubeContext;
      this.cache.terminals = [];
      this.cache.multiLog = null;
      this.forceUpdate();
    } else {
      // Update cache
      for (let i = 0; i < this.cache.terminals.length; i++) {
        const selectedPod =
          this.props.podList && this.props.podList.items.find((pod) => this.cache.terminals[i].pod === pod.metadata.name);
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

    if (this.props.selected !== this.selected) {
      this.select(this.props.selected);
      this.forceUpdate();
    }
  }
}

export default withDevSpaceConfig(TerminalCache);
