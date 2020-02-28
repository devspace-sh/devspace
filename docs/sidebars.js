/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

module.exports = {
  adminSidebar: [
    {
      type: 'doc',
      id: 'introduction',
    },
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/initialize-project',
        'getting-started/deployment',
        'getting-started/development',
        'getting-started/next-steps',
      ],
    },
    {
      type: 'category',
      label: 'Configuration',
      items: [
        'configuration/reference',
        {
          type: 'category',
          label: 'images',
          items: [
            'configuration/images/basics',
            'configuration/images/image-tagging',
            'configuration/images/dockerfile-context',
            'configuration/images/entrypoint-cmd',
            'configuration/images/pull-secrets',
            {
              type: 'category',
              label: 'build',
              items: [
                'configuration/images/docker',
                'configuration/images/kaniko',
                'configuration/images/custom',
                'configuration/images/disabled',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'deployments',
          items: [
            'configuration/deployments/basics',
            'configuration/deployments/helm-charts',
            'configuration/deployments/kubernetes-manifests',
            'configuration/deployments/kustomizations',
          ],
        },
        {
          type: 'category',
          label: 'dev',
          items: [
            'configuration/development/basics',
            'configuration/development/open-links',
            'configuration/development/port-forwarding',
            'configuration/development/file-synchronization',
            'configuration/development/auto-reloading',
            'configuration/development/log-streaming',
            'configuration/development/interactive-mode',
          ],
        },
        {
          type: 'category',
          label: 'dependencies',
          items: [
            'configuration/dependencies/basics',
            'configuration/dependencies/git-repository',
            'configuration/dependencies/local-folder',
          ],
        },
        {
          type: 'category',
          label: 'vars',
          items: [
            'configuration/variables/basics',
            'configuration/variables/source-env',
            'configuration/variables/source-input',
          ],
        },
        {
          type: 'category',
          label: 'profiles',
          items: [
            'configuration/profiles/basics',
            'configuration/profiles/patches',
            'configuration/profiles/replace',
          ],
        },
        'configuration/commands/basics',
        'configuration/hooks/basics',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'guides/basics',
        'guides/localhost-ui',
        'guides/networking-domains',
        'guides/file-synchronization',
        'guides/interactive-mode',
        'guides/ci-cd-integration',
      ],
    },
    {
      type: 'category',
      label: 'Tutorials',
      items: [
        {
          type: 'link',
          label: '↗️ Ruby on Rails',
          href: 'https://devspace.cloud/blog/2019/10/21/deploy-ruby-on-rails-to-kubernetes',
        },
        {
          type: 'link',
          label: '↗️ Python Django',
          href: 'https://devspace.cloud/blog/2019/10/18/deploy-django-to-kubernetes',
        },
        {
          type: 'link',
          label: '↗️ PHP Laravel',
          href: 'https://devspace.cloud/blog/2019/10/16/deploy-laravel-to-kubernetes',
        },
        {
          type: 'category',
          label: '↗️ Node / JavaScript',
          items: [
            {
              type: 'link',
              label: '↗️ Express.js',
              href: 'https://devspace.cloud/blog/2019/10/15/deploy-express.js-server-to-kubernetes',
            },
            {
              type: 'link',
              label: '↗️ React.js',
              href: 'https://devspace.cloud/blog/2019/03/07/deploy-react-js-to-kubernetes',
            },
            {
              type: 'link',
              label: '↗️ Vue.js',
              href: 'https://devspace.cloud/blog/2019/09/30/deploy-vue-js-to-kubernetes',
            },
          ]
        },
      ],
    },
    {
      type: 'category',
      label: 'Best Practices',
      items: [
        'best-practices/image-building',
        'best-practices/dev-staging-production',
        'best-practices/remote-debugging',
        'best-practices/community-projects',
      ],
    },
    {
      type: 'category',
      label: 'CLI Commands',
      items: [],
    },
    {
      type: 'link',
      label: '↗️ Component Chart',
      href: 'https://devspace.sh/component-chart/docs/introduction',
    },
    {
      type: 'link',
      label: '↗️ DevSpace Cloud',
      href: 'https://devspace.cloud/cloud/docs/introduction',
    },
  ],
};
