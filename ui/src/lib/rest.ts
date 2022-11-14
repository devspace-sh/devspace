export const ApiHostname = () => {
  if (location.port === '3000') {
    return 'localhost:8090';
  } else if (!location.port) {
    return location.hostname;
  }

  return location.hostname + ':' + location.port;
};

export const ApiWebsocketProtocol = () => {
  if (location.protocol === 'https:') {
    return 'wss';
  } else {
    return 'ws';
  }
}
