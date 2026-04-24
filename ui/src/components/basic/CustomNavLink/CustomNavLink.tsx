import React from 'react';
import { NavLink, NavLinkProps } from 'react-router-dom';

interface Props extends Omit<NavLinkProps, 'className'> {
  activeClassName?: string;
  className?: string;
}

const CustomNavLink = ({ activeClassName, className, end, to, ...props }: Props) => {
  const getClassName = ({ isActive }: { isActive: boolean }) =>
    [className, isActive ? activeClassName : undefined].filter(Boolean).join(' ');
  const isEnd = typeof end === 'boolean' ? end : false;

  return <NavLink {...props} className={getClassName} end={isEnd} to={to} />;
};

export default CustomNavLink;
