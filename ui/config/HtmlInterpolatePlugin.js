'use strict';

class HtmlInterpolatePlugin {
  constructor(replacements) {
    this.replacements = replacements;
  }

  apply(compiler) {
    compiler.hooks.compilation.tap('HtmlInterpolatePlugin', compilation => {
      const HtmlWebpackPlugin = require('html-webpack-plugin');
      const hooks = HtmlWebpackPlugin.getHooks(compilation);

      hooks.beforeEmit.tap('HtmlInterpolatePlugin', data => {
        data.html = Object.keys(this.replacements).reduce(
          (html, key) => html.replace(new RegExp(`%${key}%`, 'g'), this.replacements[key]),
          data.html
        );
        return data;
      });
    });
  }
}

module.exports = HtmlInterpolatePlugin;
