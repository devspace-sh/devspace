// prettier-ignore
import React from 'react';
import { Route, Switch, BrowserRouter } from 'react-router-dom';
import Logs from 'pages/logs';
import ConditionalRoute from 'components/advanced/ConditionalRoute/ConditionalRoute';

interface Props {}

const Routes = (_: Props) => {
  return (
    <BrowserRouter>
      <Switch>
        <Route exact path="/logs" component={Logs} />
        <ConditionalRoute exact path="/" redirectTo="/logs" when={true} component={Logs} />
        <Route render={() => <h1>Page not found</h1>} />
      </Switch>
    </BrowserRouter>
  );
};

export default Routes;
