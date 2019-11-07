import React from 'react';
import styles from './Breadcrumb.module.scss';
import Arrow from 'images/breadcrumb-arrow.svg';
import { withRouter, RouteComponentProps } from 'react-router';
import { Link } from 'react-router-dom';
import ErrorBoundary from 'components/basic/ErrorBoundary/ErrorBoundary';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';

const capitalize = (s: string) => {
  if (typeof s !== 'string') return '';
  return s.charAt(0).toUpperCase() + s.slice(1);
};

interface Props extends RouteComponentProps, DevSpaceConfigContext {}
interface State {}

class Breadcrumb extends React.Component<Props, State> {
  isComponentMounted: boolean = false;

  componentDidMount() {
    this.isComponentMounted = true;
  }

  componentWillUnmount() {
    this.isComponentMounted = false;
  }

  renderBreadcrumb = () => {
    const arrow = <img src={Arrow} />;

    // Removes empty string "" because path starts with "/"
    const crumbs = this.props.match.url.split('/').slice(1);
    const crumbsWithIds = crumbs.slice(0, crumbs.length - 1);
    const params = this.props.match.path.split('/').slice(1);
    const crumbsWithParams = params.slice(0, params.length - 1);

    if (crumbs.length === 1) {
      return (
        <span className={styles.crumb}>
          {arrow}
          <span className={styles.text}>{capitalize(crumbs[0])}</span>
        </span>
      );
    }

    return [
      crumbsWithParams.map((crumb: string, idx: number) => {
        const isLastOne = idx === crumbsWithParams.length - 1;
        const shouldNotBeLink = false;

        if (isLastOne || shouldNotBeLink) {
          return (
            <span className={styles.crumb} key={idx}>
              {arrow}
              <span className={styles.text}>{capitalize(crumb)}</span>
            </span>
          );
        }

        return (
          <span className={styles.crumb} key={idx}>
            {arrow}
            <Link to={'/' + crumbsWithIds.slice(0, idx + 1).join('/')}>{capitalize(crumb)}</Link>
          </span>
        );
      }),
    ];
  };

  renderPrefix = () => {
    if (!this.props.devSpaceConfig.workingDirectory || !this.props.devSpaceConfig.config) {
      return 'DevSpace';
    } else {
      const wd = this.props.devSpaceConfig.workingDirectory;
      // Unix
      const lastIdxOfSlash = wd.lastIndexOf('/');
      // Windows
      const lastIdxOfBackSlash = wd.lastIndexOf('\\');

      if (lastIdxOfSlash !== -1) {
        return './' + wd.slice(lastIdxOfSlash + 1);
      } else {
        return './' + wd.slice(lastIdxOfBackSlash + 1);
      }
    }
  };

  renderRoute() {
    return (
      <React.Fragment>
        <div className={styles['account-selector']}>{this.renderPrefix()}</div>
        <div className={styles.crumbs}>{this.renderBreadcrumb()}</div>
      </React.Fragment>
    );
  }

  render() {
    return (
      <ErrorBoundary>
        <div className={styles.breadcrumb}>{this.renderRoute()}</div>
      </ErrorBoundary>
    );
  }
}

export default withRouter(withDevSpaceConfig(Breadcrumb));
