import React from 'react';
import { NavLink, NavLinkProps } from 'react-router-dom';
import { formatURL } from 'lib/utils';

const CustomNavLink = (props: NavLinkProps) => (
  <NavLink
    isActive={(_, { pathname }) =>
      pathname.startsWith(
        typeof props.to === 'object' ? formatURL(props.to.pathname) : formatURL(props.to.toString().replace(/\?.+$/, ''))
      )
    }
    {...props}
  />
);

export default CustomNavLink;
