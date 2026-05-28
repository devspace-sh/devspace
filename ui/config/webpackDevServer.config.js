'use strict';

const protocol = process.env.HTTPS === 'true' ? 'https' : 'http';
const host = process.env.HOST || '0.0.0.0';

module.exports = function(proxy, allowedHost) {
  return {
    // Enable gzip compression of generated files.
    compress: true,
    // Enable hot reloading and the built-in webpack-dev-server client.
    hot: true,
    client: {
      overlay: true,
    },
    // Enable HTTPS if the HTTPS environment variable is set to 'true'
    server: protocol,
    host: host,
    allowedHosts: allowedHost ? [allowedHost] : 'auto',
    historyApiFallback: {
      // Paths with dots should still use the history fallback.
      // See https://github.com/facebookincubator/create-react-app/issues/387.
      disableDotRule: true,
    },
    proxy,
  };
};
