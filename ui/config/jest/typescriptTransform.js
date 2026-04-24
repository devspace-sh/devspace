// Copyright 2004-present Facebook. All Rights Reserved.

'use strict';

const path = require('path');
const typescript = require('typescript');

const configPath = path.resolve(__dirname, '../../tsconfig.test.json');
const { config } = typescript.readConfigFile(configPath, typescript.sys.readFile);
const parsedConfig = typescript.parseJsonConfigFileContent(config, typescript.sys, path.dirname(configPath));

module.exports = {
  process(src, filename) {
    const result = typescript.transpileModule(src, {
      compilerOptions: {
        ...parsedConfig.options,
        sourceMap: false,
      },
      fileName: filename,
    });

    return {
      code: result.outputText,
    };
  },
};
