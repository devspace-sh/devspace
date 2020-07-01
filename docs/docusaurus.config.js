__webpack_public_path__ = "/cli/"

const versions = require('./versions.json');

module.exports = {
  title: 'DevSpace CLI | Documentation',
  tagline: 'The tagline of my site',
  url: 'https://devspace.sh',
  baseUrl: __webpack_public_path__,
  favicon: '/img/favicon.png',
  organizationName: 'devspace-cloud', // Usually your GitHub org/user name.
  projectName: 'devspace', // Usually your repo name.
  themeConfig: {
    disableDarkMode: true,
    navbar: {
      logo: {
        alt: 'DevSpace',
        src: '/img/logo-devspace.svg',
        href: 'https://devspace.sh/',
        target: '_self'
      },
      links: [
        {
          to: 'versions',
          label: `${versions[0]}`,
          position: 'left',
          className: 'version-link'
        },
        {
          href: 'https://devspace.sh/',
          label: 'Website',
          position: 'left',
          target: '_self'
        },
        {
          href: __webpack_public_path__ + 'docs/' + (process.env.NODE_ENV == 'production' ? '' : 'next/') + 'introduction',
          label: 'Docs',
          position: 'left',
          target: '_self'
        },
        {
          href: 'https://devspace.cloud/blog',
          label: 'Blog',
          position: 'left'
        },
        {
          href: 'https://slack.k8s.io/#devspace',
          className: 'slack-link',
          'aria-label': 'Slack',
          position: 'right',
        },
        {
          href: 'https://github.com/devspace-cloud/devspace',
          className: 'github-link',
          'aria-label': 'GitHub',
          position: 'right',
        },
      ],
    },
    algolia: {
      apiKey: "b9533b52dde7e23272dbd4211435c070",
      indexName: "devspace-cli",
      placeholder: "Search...",
      algoliaOptions: {},
    },
    footer: {
      style: 'light',
      links: [],
      copyright: `Copyright © ${new Date().getFullYear()} DevSpace Authors`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          path: 'pages',
          routeBasePath: 'docs',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl:
            'https://github.com/devspace-cloud/devspace/edit/master/docs/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
  plugins: [
    [
      require.resolve('docusaurus-gtm-plugin'),
      {
        id: 'GTM-5KKTMWJ',
      }
    ]
  ],
  scripts: [
    {
      src:
        'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
      async: true,
    },
    {
      src:
        'https://devspace.sh/docs.js',
      async: true,
    },
  ],
};
