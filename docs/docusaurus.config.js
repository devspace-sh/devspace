__webpack_public_path__ = "/docs/"

const resolveGlob = require('resolve-glob');

module.exports = {
  title: 'DevSpace | Documentation',
  tagline: 'The tagline of my site',
  url: 'https://devspace.sh',
  baseUrl: __webpack_public_path__,
  markdown: {
    mdx1Compat: {
      headingIds: true,
    },
  },
  favicon: '/img/favicon.png',
  organizationName: 'loft-sh', // Usually your GitHub org/user name.
  projectName: 'devspace', // Usually your repo name.
  themeConfig: {
    colorMode: {
      disableSwitch: true
    },
    navbar: {
      //hideOnScroll: true,
      logo: {
        alt: 'DevSpace',
        src: '/media/logos/devspace-logo-primary.svg',
        href: 'https://devspace.sh/',
        target: '_self'
      },
      items: [
        {
            type: 'docsVersionDropdown',
            position: 'left',
        },
        {
            href: 'https://devspace.sh/',
            label: 'Website',
            position: 'left',
            target: '_self'
        },
        {
            href: 'https://loft.sh/blog/tags/devspace',
            label: 'Blog',
            position: 'left',
            target: '_self'
        },
        {
            href: 'https://slack.loft.sh/',
            className: 'slack-link',
            'aria-label': 'Slack',
            position: 'right',
        },
        {
            href: 'https://github.com/loft-sh/devspace',
            className: 'github-link',
            'aria-label': 'GitHub',
            position: 'right',
        },
      ],
    },
    algolia: {
      apiKey: "9396b07e4ad34e90394fbfe79695d88d",
      appId: "L1ZH1CZBMP",
      indexName: "devspace-cli",
      placeholder: "Search...",
      algoliaOptions: {},
      contextualSearch: true,
    },
    footer: {
      style: 'light',
      links: [],
      copyright: `Copyright © DevSpace Authors <br/>DevSpace is an open-source project originally created by <a href="https://loft.sh/">Loft Labs, Inc.</a>`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          path: 'pages',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          showLastUpdateTime: true,
          editUrl: 'https://github.com/loft-sh/devspace/edit/main/docs/',
          lastVersion: "current",
          versions: {
              current: {
                  label: "6.x (Latest)",
                  path: ""
              }
          },
      },
        theme: {
          customCss: resolveGlob.sync(['./src/css/**/*.scss']),
        },
      },
    ],
    [
      'redocusaurus',
      {
          specs: [
              {
                  spec: 'schemas/config-openapi.json',
              },
          ],
          theme: {
              primaryColor: '#00bdff',
              redocOptions: {
                  hideDownloadButton: false,
                  disableSearch: true,
                  colors: {
                      border: {
                          dark: '#00bdff',
                          light: '#00bdff',
                      }
                  }
              },
          },
      },
    ],
  ],
  themes: [
      '@saucelabs/theme-github-codeblock'
  ],
  plugins: [
      'docusaurus-plugin-sass',
      'plugin-image-zoom',
  ],
  scripts: [
    {
        src: 'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
        async: true,
    },
  ],
  clientModules: resolveGlob.sync(['./src/js/**/*.js']),
};
