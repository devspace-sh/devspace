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
      collapsed: false,
      items: [
        {
          type: 'doc',
          id: 'quickstart',
        },
        {
          type: 'category',
          label: 'Full Guide',
          items: [
            'getting-started/installation',
            'getting-started/initialize-project',
            'getting-started/development',
            'getting-started/deployment',
            'getting-started/cleanup',
            'getting-started/next-steps',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Configuration',
      collapsed: false,
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
            'configuration/images/append-dockerfile-instructions',
            'configuration/images/inject-restart-helper',
            'configuration/images/rebuild-strategy',
            'configuration/images/pull-secrets',
            {
              type: 'category',
              label: 'build',
              items: [
                'configuration/images/docker',
                'configuration/images/buildkit',
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
            {
              type: 'link',
              label: '↗️ Component Chart',
              href: 'https://devspace.sh/component-chart/docs',
            },
          ],
        },
        {
          type: 'category',
          label: 'dev',
          items: [
            'configuration/development/basics',
            'configuration/development/port-forwarding',
            'configuration/development/reverse-port-forwarding',
            'configuration/development/open-links',
            'configuration/development/file-synchronization',
            'configuration/development/terminal',
            'configuration/development/log-streaming',
            'configuration/development/replace-pods',
            'configuration/development/auto-reloading',
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
            'configuration/variables/source-command',
            'configuration/variables/source-none',
          ],
        },
        {
          type: 'category',
          label: 'profiles',
          items: [
            'configuration/profiles/basics',
            'configuration/profiles/replace',
            'configuration/profiles/merge',
            'configuration/profiles/patches',
            'configuration/profiles/parents',
            'configuration/profiles/activation',
          ],
        },
        'configuration/pullSecrets/basics',
        'configuration/commands/basics',
        'configuration/hooks/basics',
        'configuration/require/basics',
        'configuration/expressions',
        'configuration/env-file',
      ],
    },
    {
      type: 'category',
      label: 'Guides & Best Practices',
      items: [
        'guides/basics',
        'guides/localhost-ui',
        'guides/networking-domains',
        'guides/file-synchronization',
        'guides/ci-cd-integration',
        'guides/dev-staging-production',
        'guides/image-building',
        'guides/plugins',
        'guides/remote-debugging',
        'guides/community-projects',
      ],
    },
    {
      type: 'category',
      label: 'Tutorials',
      items: [
        {
          type: 'link',
          label: 'Ruby on Rails',
          href: 'https://devspace.cloud/blog/2019/10/21/deploy-ruby-on-rails-to-kubernetes',
        },
        {
          type: 'link',
          label: 'Python Django',
          href: 'https://devspace.cloud/blog/2019/10/18/deploy-django-to-kubernetes',
        },
        {
          type: 'link',
          label: 'PHP Laravel',
          href: 'https://devspace.cloud/blog/2019/10/16/deploy-laravel-to-kubernetes',
        },
        {
          type: 'category',
          label: 'Node / JavaScript',
          items: [
            {
              type: 'link',
              label: 'Express.js',
              href: 'https://devspace.cloud/blog/2019/10/15/deploy-express.js-server-to-kubernetes',
            },
            {
              type: 'link',
              label: 'React.js',
              href: 'https://devspace.cloud/blog/2019/03/07/deploy-react-js-to-kubernetes',
            },
            {
              type: 'link',
              label: 'Vue.js',
              href: 'https://devspace.cloud/blog/2019/09/30/deploy-vue-js-to-kubernetes',
            },
          ]
        },
      ],
    },
    {
      type: 'category',
      label: 'CLI Commands',
      items: [
        {
          type: "category",
          label: "devspace add",
          items: [
            "commands/devspace_add_plugin",
          ]
        },
        "commands/devspace_analyze",
        "commands/devspace_attach",
        "commands/devspace_build",
        "commands/devspace_cleanup_images",
        "commands/devspace_deploy",
        "commands/devspace_dev",
        "commands/devspace_enter",
        "commands/devspace_init",
        {
          type: "category",
          label: "devspace list",
          items: [
            "commands/devspace_list_commands",
            "commands/devspace_list_contexts",
            "commands/devspace_list_deployments",
            "commands/devspace_list_namespaces",
            "commands/devspace_list_plugins",
            "commands/devspace_list_ports",
            "commands/devspace_list_profiles",
            "commands/devspace_list_sync",
            "commands/devspace_list_vars"
          ]
        },
        "commands/devspace_logs",
        "commands/devspace_open",
        "commands/devspace_print",
        "commands/devspace_purge",
        {
          type: "category",
          label: "devspace remove",
          items: [
            "commands/devspace_remove_context",
            "commands/devspace_remove_plugin"
          ]
        },
        "commands/devspace_render",
        {
          type: "category",
          label: "devspace reset",
          items: [
            "commands/devspace_reset_dependencies",
            "commands/devspace_reset_vars"
          ]
        },
        "commands/devspace_run",
        {
          type: "category",
          label: "devspace set",
          items: [
            "commands/devspace_set_var"
          ]
        },
        "commands/devspace_sync",
        "commands/devspace_ui",
        {
          type: "category",
          label: "devspace update",
          items: [
            "commands/devspace_update_plugin",
            "commands/devspace_update_dependencies"
          ]
        },
        "commands/devspace_upgrade",
        {
          type: "category",
          label: "devspace use",
          items: [
            "commands/devspace_use_context",
            "commands/devspace_use_namespace",
            "commands/devspace_use_profile"
          ]
        }
      ],
    },
    {
      type: 'link',
      label: '↗️ Component Chart',
      href: 'https://devspace.sh/component-chart/docs',
    },
    {
      type: 'link',
      label: '↗️ Open-Source by Loft Labs',
      href: 'https://loft.sh/',
    },
  ],
};
