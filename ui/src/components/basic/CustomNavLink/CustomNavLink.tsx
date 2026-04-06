import React from 'react';
import { NavLink, NavLinkProps } from 'react-router-dom';

interface Props extends NavLinkProps {
  activeClassName?: string;
}

const CustomNavLink = ({ activeClassName, className, ...props }: Props) => (
  <NavLink
    {...props}
    className={({ isActive }) => [className, isActive ? activeClassName : ''].filter(Boolean).join(' ')}
  />
);

export default CustomNavLink;
