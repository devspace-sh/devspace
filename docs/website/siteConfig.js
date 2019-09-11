/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

// See https://docusaurus.io/docs/site-config for all the possible
// site configuration options.

const siteConfig = {
  projectName: "devspace-docs",
  title: "DevSpace Documentation", // Title for your website.
  tagline: "A website for testing",
  url: "https://devspace.cloud", // Your website URL
  baseUrl: "/",

  // For no header links in the top nav bar -> headerLinks: [],
  headerLinks: [
    { href: "https://devspace.cloud/pricing/", label: "" },
    { href: "https://devspace.cloud/about/", label: "About" },
    { doc: "introduction", label: "Docs" },
    { href: "https://devspace.cloud/help/", label: "Help" },
    { href: "https://app.devspace.cloud/", label: "Login" },
    { href: "#", label: "" },
    { search: true }
  ],

  /* path to images for header/footer */
  headerIcon: "img/devspace-logo.svg",
  footerIcon: "img/devspace-logo.svg",
  favicon: "img/favicon.png",

  // Open Graph and Twitter card images.
  ogImage: "img/devspace-logo.svg",
  twitterImage: "img/devspace-logo.svg",

  /* Colors for website */
  colors: {
    primaryColor: "#223366",
    secondaryColor: "#00BDFF"
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
    theme: "atom-one-dark"
  },

  // Add custom scripts here that would be placed in <script> tags.
  scripts: [
    "https://buttons.github.io/buttons.js",
    "https://cdnjs.cloudflare.com/ajax/libs/clipboard.js/2.0.0/clipboard.min.js",
    "/js/code-block-buttons.js",
    "/js/code-block-line-numbers.js",
    "/js/responsive-menu.js",
    "/js/tag-manager.js",
    "/js/onpage-nav.js"
  ],

  stylesheets: [
    "https://fonts.googleapis.com/css?family=Play|Kanit:400"
  ],

  // On page navigation for the current documentation page.
  onPageNav: "separate",
  // No .html extensions for paths.
  cleanUrl: true,
  disableHeaderTitle: true,
  disableTitleTagline: true,
  // Show documentation's last contributor's name.
  // enableUpdateBy: true,

  // Show documentation's last update time.
  enableUpdateTime: true,

  customDocsPath: "pages",
  docsSideNavCollapsible: true,
  editUrl: "https://github.com/devspace-cloud/devspace/edit/master/docs/pages/",

  algolia: {
    apiKey: "4339e8c4d6313d53209b996a35e7c0d5",
    indexName: "devspace",
    placeholder: "Search...",
    algoliaOptions: {}
  }
};

module.exports = siteConfig;
