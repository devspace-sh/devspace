/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require("react");

class Footer extends React.Component {
  docUrl(doc, language) {
    const baseUrl = this.props.config.baseUrl;
    const docsUrl = this.props.config.docsUrl;
    const docsPart = `${docsUrl ? `${docsUrl}/` : ""}`;
    const langPart = `${language ? `${language}/` : ""}`;
    return `${baseUrl}${docsPart}${langPart}${doc}`;
  }

  pageUrl(doc, language) {
    const baseUrl = this.props.config.baseUrl;
    return baseUrl + (language ? `${language}/` : "") + doc;
  }

  render() {
    let chatAndAnalytics = "";

    try {
      const Chat = require("@devspace/react-components").Chat;
      const Analytics = require("@devspace/react-components").Analytics;

      chatAndAnalytics = (
        <div>
          <Chat />
          <Analytics />
        </div>
      );
    } catch (e) {}

    return (
      <footer className="nav-footer" id="footer">
        <script type="text/javascript" dangerouslySetInnerHTML={{__html: `
        var versionMeta = document.querySelector("head > meta[name='docsearch:version']");

        if (versionMeta) {
          document.querySelector("body").setAttribute("data-version", versionMeta.getAttribute("content"));
        }

        if (location.hostname == "devspace.cloud") {
          document.querySelector(".headerWrapper > header > a:nth-child(2)").setAttribute("href", "/docs/versions");
        }
        `}}>
        </script>

        {chatAndAnalytics}

        <noscript>
          <iframe
            src="https://www.googletagmanager.com/ns.html?id=GTM-KM6KSWG"
            height="0"
            width="0"
            style={{ display: "none", visibility: "hidden" }}
          />
        </noscript>
      </footer>
    );
  }
}

module.exports = Footer;
