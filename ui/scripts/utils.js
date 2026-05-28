'use strict';

const childProcess = require('child_process');
const fs = require('fs');
const net = require('net');
const path = require('path');
const zlib = require('zlib');

function checkRequiredFiles(files) {
  const missing = files.filter(file => !fs.existsSync(file));
  missing.forEach(file => console.log(`Required file not found: ${file}`));
  return missing.length === 0;
}

function clearConsole() {
  process.stdout.write('\x1bc');
}

function formatWebpackMessages(json) {
  return {
    errors: (json.errors || []).map(formatWebpackMessage),
    warnings: (json.warnings || []).map(formatWebpackMessage),
  };
}

function formatWebpackMessage(message) {
  if (typeof message === 'string') {
    return message;
  }
  return message.message || message.details || String(message);
}

function printBuildError(err) {
  console.log(err && err.stack ? err.stack : err);
}

function collectFiles(dir) {
  if (!fs.existsSync(dir)) {
    return [];
  }

  return fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const file = path.join(dir, entry.name);
    return entry.isDirectory() ? collectFiles(file) : [file];
  });
}

function fileSize(file) {
  if (/\.(js|css)$/.test(file)) {
    return zlib.gzipSync(fs.readFileSync(file)).length;
  }

  return fs.statSync(file).size;
}

function measureFileSizesBeforeBuild(buildFolder) {
  return Promise.resolve(
    new Map(collectFiles(buildFolder).map(file => [path.relative(buildFolder, file), fileSize(file)]))
  );
}

function formatBytes(bytes) {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} kB`;
  }
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

function printFileSizesAfterBuild(stats, previousFileSizes, buildFolder, maxBundleGzipSize, maxChunkGzipSize) {
  const assets = stats
    .toJson({ all: false, assets: true })
    .assets.filter(asset => /\.(js|css)$/.test(asset.name))
    .map(asset => {
      const file = path.join(buildFolder, asset.name);
      const gzipSize = fs.existsSync(file) ? zlib.gzipSync(fs.readFileSync(file)).length : 0;
      const previousSize = previousFileSizes.get(asset.name);
      const diff = previousSize ? gzipSize - previousSize : 0;
      const diffText = diff ? ` (${diff > 0 ? '+' : ''}${formatBytes(diff)})` : '';
      const limit = asset.name.endsWith('.chunk.js') ? maxChunkGzipSize : maxBundleGzipSize;
      const warning = gzipSize > limit ? ' [big]' : '';
      return `  ${formatBytes(gzipSize)}${diffText}${warning}  ${asset.name}`;
    });

  assets.forEach(asset => console.log(asset));
}

function printHostingInstructions(appPackage, publicUrl, publicPath, buildFolder) {
  console.log(`The project was built assuming it is hosted at ${publicPath || 'the server root'}.`);
  if (!appPackage.homepage) {
    console.log('You can control this with the homepage field in your package.json.');
  }
  console.log();
  console.log(`The ${buildFolder} folder is ready to be deployed.`);
}

function canUsePort(host, port) {
  return new Promise(resolve => {
    const server = net.createServer();
    server.once('error', () => resolve(false));
    server.once('listening', () => {
      server.close(() => resolve(true));
    });
    server.listen(port, host);
  });
}

async function choosePort(host, defaultPort) {
  for (let port = defaultPort; port < defaultPort + 10; port += 1) {
    if (await canUsePort(host, port)) {
      return port;
    }
  }
  return null;
}

function prepareUrls(protocol, host, port) {
  const browserHost = host === '0.0.0.0' ? 'localhost' : host;
  const localUrlForBrowser = `${protocol}://${browserHost}:${port}/`;
  return {
    localUrlForBrowser,
    lanUrlForConfig: host,
  };
}

function prepareProxy(proxySetting) {
  if (!proxySetting) {
    return undefined;
  }

  return [
    {
      context: ['/api'],
      target: proxySetting,
      changeOrigin: true,
      secure: false,
      ws: true,
    },
  ];
}

function openBrowser(url) {
  if (process.env.BROWSER === 'none') {
    return false;
  }

  const browser = process.env.BROWSER;
  const command = browser || (process.platform === 'darwin' ? 'open' : process.platform === 'win32' ? 'cmd' : 'xdg-open');
  const args = browser ? [url] : process.platform === 'win32' ? ['/c', 'start', '""', url] : [url];
  const child = childProcess.spawn(command, args, { detached: true, stdio: 'ignore' });
  child.unref();
  return true;
}

module.exports = {
  checkRequiredFiles,
  choosePort,
  clearConsole,
  formatWebpackMessages,
  measureFileSizesBeforeBuild,
  openBrowser,
  prepareProxy,
  prepareUrls,
  printBuildError,
  printFileSizesAfterBuild,
  printHostingInstructions,
};
