// prettier-ignore
import React from 'react';
import { BrowserRouter, Navigate, Route, Routes as RouterRoutes } from 'react-router-dom';
import LogsContainers from 'pages/logs/containers';
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
        <Route path="/" element={<Navigate replace to="/logs/containers" />} />
        <Route path="*" element={<h1>Page not found</h1>} />
      </RouterRoutes>
    </BrowserRouter>
  );
};

export default Routes;
