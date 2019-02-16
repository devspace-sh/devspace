/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// See https://docusaurus.io/docs/site-config for all the possible
// site configuration options.

const siteConfig = {
  title: 'DevSpace Documentation', // Title for your website.
  tagline: 'A website for testing',
  url: 'https://devspace.cloud', // Your website URL
  baseUrl: '/',

  // For no header links in the top nav bar -> headerLinks: [],
  headerLinks: [
    {href: 'http://devspace.cloud/products', label: 'Products'},
    {doc: 'getting-started/installation', label: 'Docs'},
    {href: 'http://devspace.cloud/', label: 'Help'},
  ],

  /* path to images for header/footer */
  headerIcon: 'img/docusaurus.svg',
  footerIcon: 'img/docusaurus.svg',
  favicon: 'img/favicon.png',

  // Open Graph and Twitter card images.
  ogImage: 'img/docusaurus.png',
  twitterImage: 'img/docusaurus.png',

  /* Colors for website */
  colors: {
    primaryColor: '#223366',
    secondaryColor: '#00BDFF',
  },

  /* Custom fonts for website */
  /*
  fonts: {
    myFont: [
      "Times New Roman",
      "Serif"
    ],
    myOtherFont: [
      "-apple-system",
      "system-ui"
    ]
  },
  */

  // This copyright info is used in /core/Footer.js and blog RSS/Atom feeds.
  copyright: `Copyright © ${new Date().getFullYear()} DevSpace`,

  highlight: {
    // Highlight.js theme to use for syntax highlighting in code blocks.
    theme: 'atom-one-dark',
  },

  // Add custom scripts here that would be placed in <script> tags.
  scripts: [
    'https://buttons.github.io/buttons.js',
    'https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js',
    '/js/code-block-buttons.js',
  ],

  stylesheets: [
    '/css/code-block-buttons.css'
  ],

  // On page navigation for the current documentation page.
  onPageNav: 'separate',
  // No .html extensions for paths.
  cleanUrl: true,

  // Show documentation's last contributor's name.
  // enableUpdateBy: true,

  // Show documentation's last update time.
  enableUpdateTime: true,

  customDocsPath: 'pages',
  docsSideNavCollapsible: true,
};

module.exports = siteConfig;
