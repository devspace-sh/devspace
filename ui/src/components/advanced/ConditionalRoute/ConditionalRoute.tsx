import React from 'react';
import { Navigate } from 'react-router-dom';

const ConditionalRoute = ({
  component: Component,
  redirectTo,
  when,
  ...rest
}: {
  component?: any;
  redirectTo: string;
  when: boolean;
  condition?: string;
}) => {
  if (when) {
    return <Navigate to={redirectTo} replace />;
  } else if (Component) {
    return <Component {...rest} />;
  }

  return null;
};

export default ConditionalRoute;
