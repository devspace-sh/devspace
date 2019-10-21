import http, { IncomingMessage } from 'http';
import https from 'https';
import React from 'react';

export const formatError = (error: any): any => {
  if (!error) {
    return undefined;
  }

  return error && error.message ? error.message : '' + error;
};

export function bindParameter(fnToBind: (...args: any[]) => any, ...args: any[]) {
  return (...newArgs: any[]): any => fnToBind(...args, ...newArgs);
}

export const timeSince = (date: Date) => {
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
  let interval = Math.floor(seconds / 31536000);

  if (interval > 1) {
    return interval + ' years';
  }

  interval = Math.floor(seconds / 2592000);
  if (interval > 1) {
    return interval + ' months';
  }

  interval = Math.floor(seconds / 86400);
  if (interval > 1) {
    return interval + ' days';
  }

  interval = Math.floor(seconds / 3600);
  if (interval > 1) {
    return interval + ' hours';
  }

  interval = Math.floor(seconds / 60);
  if (interval > 1) {
    return interval + ' minutes';
  }

  return Math.floor(seconds) + ' seconds';
};

export function formatURL(url: string) {
  return url.replace(new RegExp('/+$', 'g'), '');
}

export function getHashParams() {
  const hashParams = {};
  const a = /\+/g; // Regex for replacing addition symbol with a space
  const r = /([^&;=]+)=?([^&;]*)/g;
  const d = (s: string) => decodeURIComponent(s.replace(a, ' '));
  const q = window.location.hash.substring(1);

  let e = r.exec(q);
  while (e) {
    hashParams[d(e[1])] = d(e[2]);
    e = r.exec(q);
  }

  return hashParams;
}

const fallbackCopyTextToClipboard = (text: any) => {
  const textArea = document.createElement('textarea');
  textArea.value = text;
  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();

  try {
    const successful = document.execCommand('copy');
    const msg = successful ? 'successful' : 'unsuccessful';
    console.log('Fallback: Copying text command was ' + msg);
  } catch (err) {
    console.error('Fallback: Oops, unable to copy', err);
  }

  document.body.removeChild(textArea);
};
export function copyToClipboard(text: string) {
  if (!(navigator as any).clipboard) {
    fallbackCopyTextToClipboard(text);
    return;
  }

  (navigator as any).clipboard.writeText(text).then(
    () => {
      console.log('Async: Copying to clipboard was successful!');
    },
    (err: Error) => {
      console.error('Async: Could not copy text: ', err);
    }
  );
}

export const deepCopy: <T>(obj: T) => T = (obj: any) => {
  if (!obj) {
    return obj;
  }

  return JSON.parse(JSON.stringify(obj));
};

export const getUrlStatusCode = async (url: string): Promise<any> => {
  const options = { method: 'HEAD', rejectUnauthorized: false };
  const client = url.startsWith('https') ? https : http;

  return new Promise((resolve, reject) => {
    client
      .request(url, options, async (r: IncomingMessage) => {
        console.log(r.statusCode);
        if (r.statusCode === 200 || r.statusCode === 201 || r.statusCode === 202) {
          console.log('link is up!!');
          resolve({ isUp: true, statusCode: r.statusCode });
        } else {
          console.log('link is down :(');
          resolve({ isUp: false, statusCode: r.statusCode });
        }
      })
      .on('error', (err) => {
        console.log(err);
        reject();
      })
      .end();
  })
    .then((resp) => {
      return resp;
    })
    .catch(() => {
      return false;
    });
};

export const AddExtraProps = (Component: JSX.Element, extraProps: any) => {
  return <Component.type key={Component.key} {...Component.props} {...extraProps} />;
};

export const appendParamsURL = (link: string, hash?: boolean) => {
  const currentUrl = new URL(window.location.href);
  let newUrl = link + currentUrl.search;

  if (hash) newUrl += currentUrl.hash;

  return newUrl;
};

export const isTouchDevice = () => {
  const prefixes = ' -webkit- -moz- -o- -ms- '.split(' ');
  const mq = (query: any) => {
    return window.matchMedia(query).matches;
  };

  if ('ontouchstart' in window || ((window as any).DocumentTouch && document instanceof (window as any).DocumentTouch)) {
    return true;
  }

  // include the 'heartz' as a way to have a non matching MQ to help terminate the join
  // https://git.io/vznFH
  return mq(['(', prefixes.join('touch-enabled),('), 'heartz', ')'].join(''));
};

export const uuidv4 = () => {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0,
      v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
};

export const customSort = (prop: string | string[], direction: 'asc' | 'desc', arr: any[]) => {
  return arr.sort((a, b) => {
    if (typeof prop === 'string') {
      a = a[prop];
      b = b[prop];
    } else {
      a = a[prop[0]][prop[1]];
      b = b[prop[0]][prop[1]];
    }

    if (direction === 'asc') {
      if (a < b) {
        return -1;
      }
      if (a > b) {
        return 1;
      }
      return 0;
    } else {
      if (a < b) {
        return 1;
      }
      if (a > b) {
        return -1;
      }
      return 0;
    }
  });
};
