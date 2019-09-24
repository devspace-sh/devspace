/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

const React = require('react');
const CompLibrary = require('../../core/CompLibrary');
const Container = CompLibrary.Container;
const CWD = process.cwd();
const fs = require('fs')

const versions = require(`${CWD}/versions.json`);

function Versions(props) {
  const {config: siteConfig} = props;
  const latestVersion = versions[0];
  const repoUrl = `https://github.com/${siteConfig.organizationName}/${
    siteConfig.projectName
  }`;
  let firstPageOfVersion = {}

  for (let i in versions) {
    let version = versions[i];
    let versionSplit = version.split(".");
    let major = versionSplit[0].substr(1);
    let minor = versionSplit[1];
    let revision = versionSplit[2];

    loop1: 
      while (true) {
        let sidebarFile = `${CWD}/versioned_sidebars/version-v${major}.${minor}.${revision}-sidebars.json`;

        try {
          if (fs.existsSync(sidebarFile)) {
            let sidebar = require(sidebarFile);

            for (let key in sidebar) {
              for (let key2 in sidebar[key]) {
                firstPageOfVersion[version] = sidebar[key][key2][0].replace(`version-v${major}.${minor}.${revision}-`, '');
                break loop1;
              }
            }
          }
        } catch(err) {}

        if (revision > 0) {
          revision--;
        } else if (minor > 0) {
          revision = 50;
          minor--;
        } else if (major > 0) {
          revision = 50;
          minor = 50;
          major--;
        } else {
          firstPageOfVersion[version] = "introduction";
          break;
        }
      }
  }
  return (
    <div className="docMainWrapper wrapper">
      <Container className="mainContainer versionsContainer">
        <div className="post">
          <header className="postHeader">
            <h1>{siteConfig.title} Versions</h1>
          </header>
          <p></p>
          <h3 id="latest">Current version (Stable)</h3>
          <p>
            This is the documentation of the latest stable version of DevSpace.
          </p>
          <table className="versions">
            <tbody>
              <tr>
                <th>{latestVersion}</th>
                <td>
                  {/* You are supposed to change this href where appropriate
                        Example: href="<baseUrl>/docs(/:language)/:id" */}
                  <a
                    href={`${siteConfig.baseUrl}${siteConfig.docsUrl}/${
                      props.language ? props.language + '/' : ''
                    }introduction`}>
                    Documentation
                  </a>
                </td>
                <td>
                  <a href={`${repoUrl}/releases/tag/${latestVersion}`}>Release Notes</a>
                </td>
              </tr>
            </tbody>
          </table>
          <h3 id="rc">Pre-release versions</h3>
          <p>This is the work-in-progress documentation for the next release.</p>
          <table className="versions">
            <tbody>
              <tr>
                <th>master</th>
                <td>
                  {/* You are supposed to change this href where appropriate
                        Example: href="<baseUrl>/docs(/:language)/next/:id" */}
                  <a
                    href={`${siteConfig.baseUrl}${siteConfig.docsUrl}/${
                      props.language ? props.language + '/' : ''
                    }next/introduction`}>
                    Documentation
                  </a>
                </td>
                <td>
                  <a href={repoUrl}>Source Code</a>
                </td>
              </tr>
            </tbody>
          </table>
          <h3 id="archive">Past Versions</h3>
          <p>Here you can find previous versions of the documentation.</p>
          <table className="versions">
            <tbody>
              {versions.map(
                version =>
                  version !== latestVersion && (
                    <tr>
                      <th>{version}</th>
                      <td>
                        {/* You are supposed to change this href where appropriate
                        Example: href="<baseUrl>/docs(/:language)/:version/:id" */}
                        <a
                          href={`${siteConfig.baseUrl}${siteConfig.docsUrl}/${
                            props.language ? props.language + '/' : ''
                          }${version}/${firstPageOfVersion[version]}`}>
                          Documentation
                        </a>
                      </td>
                      <td>
                        <a href={`${repoUrl}/releases/tag/${version}`}>
                          Release Notes
                        </a>
                      </td>
                    </tr>
                  ),
              )}
            </tbody>
          </table>
          <p>
            You can find past versions of this project on{' '}
            <a href={repoUrl}>GitHub</a>.
          </p>
        </div>
      </Container>
    </div>
  );
}

module.exports = Versions;
