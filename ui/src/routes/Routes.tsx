// prettier-ignore
import React from 'react';
import { Route, Switch, BrowserRouter } from 'react-router-dom';
import LogsContainers from 'pages/logs/containers';
import ConditionalRoute from 'components/advanced/ConditionalRoute/ConditionalRoute';
import StackConfiguration from 'pages/stack/configuration';
import Commands from 'pages/commands/commands';

interface Props {}

const Routes = (_: Props) => {
  return (
    <BrowserRouter>
      <Switch>
        <Route exact path="/logs/containers" component={LogsContainers} />
        <Route exact path="/stack/configuration" component={StackConfiguration} />
        <Route exact path="/commands/commands" component={Commands} />
        <ConditionalRoute exact path="/" redirectTo="/logs/containers" when={true} component={LogsContainers} />
        <Route render={() => <h1>Page not found</h1>} />
      </Switch>
    </BrowserRouter>
  );
};

export default Routes;
