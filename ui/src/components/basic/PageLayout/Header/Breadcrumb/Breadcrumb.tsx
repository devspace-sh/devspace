import React from 'react';
import styles from './Breadcrumb.module.scss';
import Arrow from 'images/breadcrumb-arrow.svg';
import { Link, useLocation } from 'react-router-dom';
import ErrorBoundary from 'components/basic/ErrorBoundary/ErrorBoundary';
import withDevSpaceConfig, { DevSpaceConfigContext } from 'contexts/withDevSpaceConfig/withDevSpaceConfig';
import { formatURL } from 'lib/utils';

const capitalize = (s: string) => {
  if (typeof s !== 'string') return '';
  return s.charAt(0).toUpperCase() + s.slice(1);
};

interface Props extends DevSpaceConfigContext {}

const Breadcrumb = (props: Props) => {
  const location = useLocation();

  const renderBreadcrumb = () => {
    const arrow = <img alt="" src={Arrow} />;
    const crumbs = formatURL(location.pathname)
      .split('/')
      .filter(Boolean);
    const breadcrumbs = crumbs.length > 1 ? crumbs.slice(0, crumbs.length - 1) : crumbs;

    if (!breadcrumbs.length) {
      return null;
    }

    if (breadcrumbs.length === 1) {
      return (
        <span className={styles.crumb}>
          {arrow}
          <span className={styles.text}>{capitalize(breadcrumbs[0])}</span>
        </span>
      );
    }

    return breadcrumbs.map((crumb: string, idx: number) => {
      const isLastOne = idx === breadcrumbs.length - 1;

      if (isLastOne) {
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
          <Link to={'/' + breadcrumbs.slice(0, idx + 1).join('/')}>{capitalize(crumb)}</Link>
        </span>
      );
    });
  };

  const renderPrefix = () => {
    if (!props.devSpaceConfig.workingDirectory || !props.devSpaceConfig.config) {
      return 'DevSpace';
    } else {
      const wd = props.devSpaceConfig.workingDirectory;
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

  const renderRoute = () => {
    return (
      <React.Fragment>
        <div className={styles['account-selector']}>{renderPrefix()}</div>
        <div className={styles.crumbs}>{renderBreadcrumb()}</div>
      </React.Fragment>
    );
  };

  return (
    <ErrorBoundary>
      <div className={styles.breadcrumb}>{renderRoute()}</div>
    </ErrorBoundary>
  );
};

export default withDevSpaceConfig(Breadcrumb);
