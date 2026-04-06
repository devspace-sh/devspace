// prettier-ignore
import React from 'react';
import { Route, Routes as RouterRoutes, BrowserRouter } from 'react-router-dom';
import LogsContainers from 'pages/logs/containers';
import ConditionalRoute from 'components/advanced/ConditionalRoute/ConditionalRoute';
import StackConfiguration from 'pages/stack/configuration';
import Commands from 'pages/commands/commands';

interface Props {}

const Routes = (_: Props) => {
  return (
    <BrowserRouter>
      <RouterRoutes>
        <Route path="/logs/containers" element={<LogsContainers />} />
        <Route path="/stack/configuration" element={<StackConfiguration />} />
        <Route path="/commands/commands" element={<Commands />} />
        <Route path="/" element={<ConditionalRoute redirectTo="/logs/containers" when={true} component={LogsContainers} />} />
        <Route path="*" element={<h1>Page not found</h1>} />
      </RouterRoutes>
    </BrowserRouter>
  );
};

export default Routes;
