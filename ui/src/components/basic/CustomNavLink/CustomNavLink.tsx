import React from 'react';
import { NavLink, NavLinkProps, useLocation, useResolvedPath } from 'react-router-dom';
import { formatURL } from 'lib/utils';

interface Props extends Omit<NavLinkProps, 'className'> {
  activeClassName?: string;
  className?: string;
}

const CustomNavLink = ({ activeClassName, className, to, ...props }: Props) => {
  const location = useLocation();
  const resolvedPath = useResolvedPath(to);
  const currentPath = formatURL(location.pathname);
  const targetPath = formatURL(resolvedPath.pathname);
  const isActive = targetPath ? currentPath.startsWith(targetPath) : currentPath === targetPath;
  const classNames = [className, isActive ? activeClassName : undefined].filter(Boolean).join(' ');

  return <NavLink {...props} className={classNames} to={to} />;
};

export default CustomNavLink;
