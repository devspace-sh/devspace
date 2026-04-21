const AUTH_HEADER_NAME = 'X-DevSpace-UI-Token';
const AUTH_QUERY_PARAM = 'devspace-ui-token';
const AUTH_STORAGE_KEY = 'devspace-ui-token';

export const persistAuthTokenFromURL = () => {
  if (typeof window === 'undefined') {
    return;
  }

  const url = new URL(window.location.href);
  const token = url.searchParams.get(AUTH_QUERY_PARAM);
  if (!token) {
    return;
  }

  window.sessionStorage.setItem(AUTH_STORAGE_KEY, token);
  url.searchParams.delete(AUTH_QUERY_PARAM);
  window.history.replaceState(window.history.state, document.title, `${url.pathname}${url.search}${url.hash}`);
};

export const getStoredAuthToken = () => {
  if (typeof window === 'undefined') {
    return '';
  }

  return window.sessionStorage.getItem(AUTH_STORAGE_KEY) || '';
};

export const withAuthQuery = (url: string) => {
  const token = getStoredAuthToken();
  if (!token) {
    return url;
  }

  const parsedURL = new URL(url, window.location.href);
  parsedURL.searchParams.set(AUTH_QUERY_PARAM, token);
  return parsedURL.toString();
};

export const withAuthHeaders = (headers?: HeadersInit) => {
  const out = new Headers(headers);
  const token = getStoredAuthToken();
  if (token) {
    out.set(AUTH_HEADER_NAME, token);
  }

  return out;
};
