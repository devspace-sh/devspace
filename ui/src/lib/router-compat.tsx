import React from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

export interface RouteComponentProps<P = any> {
  history: {
    push: (to: string) => void;
    replace: (to: string) => void;
    go: (delta: number) => void;
    goBack: () => void;
    goForward: () => void;
  };
  location: any;
  match: {
    path: string;
    url: string;
    isExact: boolean;
    params: P;
  };
}

export function withRouter(Component: any) {
  return function ComponentWithRouterProp(props: any) {
    const location = useLocation();
    const navigate = useNavigate();
    const params = useParams();
    const history = {
      push: (to: string) => navigate(to),
      replace: (to: string) => navigate(to, { replace: true }),
      go: (delta: number) => navigate(delta),
      goBack: () => navigate(-1),
      goForward: () => navigate(1),
    };
    const match = {
      path: location.pathname,
      url: location.pathname,
      isExact: true,
      params,
    };

    return <Component {...props} history={history} location={location} match={match} />;
  };
}
