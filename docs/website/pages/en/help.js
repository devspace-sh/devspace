/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require('react');

const CompLibrary = require('../../core/CompLibrary.js');
const Container = CompLibrary.Container;
const GridBlock = CompLibrary.GridBlock;

const siteConfig = require(process.cwd() + '/siteConfig.js');

function docUrl(doc, language) {
  return siteConfig.baseUrl + 'docs/' + (language ? language + '/' : '') + doc;
}

class Help extends React.Component {
  render() {
    let language = this.props.language || '';
    const supportLinks = [
      {
        title: 'Quickstart Guide',
        content: `See how to [get started with your DevSpace](${docUrl('getting-started/quickstart.html', language)}).`,
      },
      {
        title: 'FAQ',
        content: 'Check out the **[Frequently Asked Questions (FAQ)](/docs/getting-started/faq)**',
      },
      {
        title: 'Further Questions?',
        content: "Feel free to open a **[new issue on GitHub](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Fquestion&title=Question:)**",
      },
    ];

    return (
      <div className="docMainWrapper wrapper">
        <Container className="mainContainer documentContainer postContainer">
          <div className="post">
            <header className="postHeader">
              <h1>Need Help?</h1>
            </header>
            <p>Follow these links for community support:</p>
            <GridBlock contents={supportLinks} layout="threeColumn" />
            <br />
            <h2>Professional Support</h2>
            <p>The DevSpace.cli is an open source project sponsored and maintained by the devspace-cloud GmbH. Our team offers DevSpace hosting as well as support and services around Kubernetes, Docker and DevSpaces. If you need professional support, get in touch with us: sales@devspace-cloud.com</p>
          </div>
        </Container>
      </div>
    );
  }
}

module.exports = Help;
