import * as React from 'react';
import * as ReactDOM from 'react-dom';
import App from 'pages/_app';
import './styles/global.scss';
import { persistAuthTokenFromURL } from './lib/auth';

persistAuthTokenFromURL();

ReactDOM.render(<App />, document.getElementById('root') as HTMLElement);
