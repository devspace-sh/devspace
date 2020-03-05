import React from 'react';
import { Route, Redirect } from 'react-router-dom';

const ConditionalRoute = ({
  component: Component,
  exact,
  path,
  redirectTo,
  when,
  condition,
  ...rest
}: {
  component?: any;
  exact?: true;
  path: string;
  redirectTo: string;
  when: boolean;
  condition?: string;
}) => {
  const renderComponent = (props: any) => <Component {...props} />;
  const redirectRoute = () => <Redirect to={redirectTo} />;

  if (when) {
    return <Route {...rest} render={redirectRoute} />;
  } else if (Component) {
    return <Route {...rest} render={renderComponent} />;
  }

  return null;
};

export default ConditionalRoute;
