__webpack_public_path__ = "/cli/"

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
      },
      links: [
        {to: 'https://devspace.sh/', label: 'Website', position: 'left'},
        {to: 'docs/introduction', label: 'Docs', position: 'left'},
        {href: 'https://devspace.cloud/blog', label: 'Blog', position: 'left'},
        {
          href: 'https://github.com/devspace-cloud/devspace',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    algolia: {
      apiKey: "4339e8c4d6313d53209b996a35e7c0d5",
      indexName: "devspace",
      placeholder: "Search...",
      algoliaOptions: {}
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
  scripts: [
    {
      src:
        'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
      async: true,
    },
  ],
};
