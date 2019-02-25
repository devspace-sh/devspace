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
    const docsUrl = this.props.config.docsUrl;
    const docsPart = `${docsUrl ? `${docsUrl}/` : ''}`;
    const langPart = `${language ? `${language}/` : ''}`;
    return `${baseUrl}${docsPart}${langPart}${doc}`;
  }

  pageUrl(doc, language) {
    const baseUrl = this.props.config.baseUrl;
    return baseUrl + (language ? `${language}/` : '') + doc;
  }

  render() {
    return (
      <footer className="nav-footer" id="footer">
        <div className="footer-container">
          <div className="devspace-company">
              <img src="/img/devspace-logo.svg" />
              DevSpace CLI and DevSpace.cloud are products developed by the covexo GmbH. 
              The terms "DevSpace" and "covexo" are registered trademarks of the covexo GmbH.
          </div>
          <div className="social-networks">
              <a className="fb-icon" href="https://www.facebook.com/covexo" target="_blank"><img src="/img/facebook-square.svg" /></a>
              <a className="twitter-icon" href="https://twitter.com/covexo" target="_blank"><img src="/img/twitter-square.svg" /></a>
              <a className="sof-icon" href="https://stackoverflow.com/questions/tagged/devspace" target="_blank"><img src="/img/stackoverflow-square.svg" /></a>
              <a className="gh-icon" href="https://github.com/covexo/devspace" target="_blank"><img src="/img/github-square.svg" /></a>
          </div>
          <div className="doc-links">
            <div className="links">
                <h5 className="title">DevSpace CLI</h5>
                <a className="link" target="_blank" href="https://github.com/covexo/devspace">GitHub Repository</a>
                <a className="link" href="/getting-started">Quickstart Guide</a>
                <a className="link" target="_blank" href="https://devspace.covexo.com/docs/cli/init.html">Command List</a>
                <a className="link" target="_blank" href="https://devspace.covexo.com/docs/getting-started/faq.html">FAQ</a>
            </div>
            <div className="links">
                <h5 className="title">DevSpace Cloud</h5>
                <a className="link" href="/products">Products</a>
                <a className="link" href="/products">Pricing</a>
                <a className="link" href="/products">Enterprise Edition</a>
            </div>
            <div className="links invisible"></div>
            <div className="links invisible"></div>
            <div className="links">
                <h5 className="title">Documentation</h5>
                <a className="link" href="/getting-started">Getting Started</a>
                <a className="link" target="_blank" href="https://docs.devspace-cloud.com/docs/configuration/config.yaml.html">Configuration</a>
                <a className="link" target="_blank" href="https://docs.devspace-cloud.com/docs/advanced/architecture.html">Architecture</a>
            </div>
            <div className="links">
                <h5 className="title">Community</h5>
                <a className="link" target="_blank" href="http://slack.devspace-cloud.com">Slack Chat</a>
                <a className="link" target="_blank" href="https://www.meetup.com/members/231546888">Meetups</a>
                <a className="link" target="_blank" href="https://github.com/covexo/devspace#contributing">Contribute</a>
            </div>
            <div className="links">
                <h5 className="title">Legal</h5>
                <a className="link" href="/terms">Terms and Conditions</a>
                <a className="link" href="/privacy-policy">Privacy Policy</a>
                <a className="link" href="/legal-notice">Legal Notice</a>
            </div>
            <div className="links"/>
          </div>
        </div>
      </footer>
    );
  }
}


module.exports = Footer;
