/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require('react');

class Footer extends React.Component {
  docUrl(doc, language) {
    const baseUrl = this.props.config.baseUrl;
    return baseUrl + 'docs/' + doc;
  }

  pageUrl(doc, language) {
    const baseUrl = this.props.config.baseUrl;
    return baseUrl + (language ? language + '/' : '') + doc;
  }

  render() {
    const currentYear = new Date().getFullYear();
    return (
      <footer className="nav-footer" id="footer">
        <section className="sitemap">
          <a href={this.props.config.baseUrl} className="nav-home">
            {this.props.config.footerIcon && (
              <img
                src={this.props.config.baseUrl + this.props.config.footerIcon}
                alt={this.props.config.title}
                width="66"
                height="58"
              />
            )}
          </a>
          <div>
            <h5>Docs</h5>
            <a href={this.docUrl('getting-started/faq.html', this.props.language)}>
              Frequently Asked Questions (FAQ)
            </a>
            <a href={this.docUrl('getting-started/quickstart.html', this.props.language)}>
              Getting Started (Quickstart Guide)
            </a>
            <a href={this.docUrl('cli/init.html', this.props.language)}>
              Command Line Interface (CLI Docs)
            </a>
            <a href={this.docUrl('configuration/dockerfile.html', this.props.language)}>
              Configuration Files
            </a>
          </div>
          <div>
            <h5>Community</h5>
            <a
              href="http://stackoverflow.com/questions/tagged/devspace"
              target="_blank"
              rel="noreferrer noopener">
              Stack Overflow
            </a>
            <a
              href="https://twitter.com/devspace-cloud"
              target="_blank"
              rel="noreferrer noopener">
              Twitter
            </a>
          </div>
          <div>
            <h5>More</h5>
            <a href="https://github.com/devspace-cloud/devspace">GitHub</a>
            <a
              className="github-button"
              href={this.props.config.repoUrl}
              data-icon="octicon-star"
              data-count-href="/devspace-cloud/devspace/stargazers"
              data-show-count={true}
              data-count-aria-label="# stargazers on GitHub"
              aria-label="Star this project on GitHub">
              Star
            </a>
          </div>
        </section>

        <a
          href="https://github.com/devspace-cloud/devspace/"
          target="_blank"
          rel="noreferrer noopener"
          className="footer-logo">
          <img
            src={this.props.config.baseUrl + 'img/devspace-logo.svg'}
            alt="DevSpace.cli Logo"
            width="300"
            height="54"
          />
        </a>
        <section className="copyright">{this.props.config.copyright}</section>
      </footer>
    );
  }
}

module.exports = Footer;
