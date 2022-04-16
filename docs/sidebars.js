module.exports = {
  adminSidebar: [
    {
      type: 'category',
      label: 'Getting Started',
      link: { type: 'doc', id: 'getting-started/introduction' },
      items: [
        'getting-started/installation',
        'getting-started/initialize-project',
        'getting-started/development',
        'getting-started/deployment',
        'getting-started/cleanup',
        'getting-started/next-steps',
      ],
    },
    {
      type: 'category',
      label: 'devspace.yaml',
      className: 'code-style',
      link: { type: 'doc', id: 'configuration/reference' },
      items: [
        'configuration/imports/basics',
        'configuration/functions/basics',
        'configuration/pipelines/basics',
        {
          type: 'category',
          label: 'images',
          link: { type: 'doc', id: 'configuration/images/basics' },
          items: [
            {
              type: 'category',
              label: '1. Build',
              link: { type: 'doc', id: 'configuration/images/build' },
              items: [
                'configuration/images/build/args',
                'configuration/images/build/multi-stage',
                'configuration/images/build/rebuild',
              ],
            },
            'configuration/images/tag',
            'configuration/images/push',
            'configuration/images/pull-secrets',
            {
              type: 'category',
              label: 'Registry Auth',
              link: { type: 'doc', id: 'configuration/images/registries/basics' },
              items: [
                'configuration/images/registries/docker-hub',
                'configuration/images/registries/github',
                'configuration/images/registries/aws',
                'configuration/images/registries/google',
                'configuration/images/registries/azure',
                'configuration/images/registries/other',
              ],
            },
            {
              type: 'category',
              label: 'Build Engines',
              link: { type: 'doc', id: 'configuration/images/build-engines/basics' },
              items: [
                'configuration/images/build-engines/docker',
                'configuration/images/build-engines/buildkit',
                'configuration/images/build-engines/kaniko',
                'configuration/images/build-engines/custom',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'deployments',
          link: { type: 'doc', id: 'configuration/deployments/basics' },
          items: [
            {
              type: 'category',
              label: 'helm',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/deployments/helm/basics' },
              items: [
                'configuration/deployments/helm/component-chart',
                'configuration/deployments/helm/local',
                'configuration/deployments/helm/remote',
              ],
            },
            {
              type: 'category',
              label: 'kubectl',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/deployments/kubectl/basics' },
              items: [
                'configuration/deployments/kubectl/manifests',
                'configuration/deployments/kubectl/kustomizations',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'dev',
          link: { type: 'doc', id: 'configuration/dev/basics' },
          items: [
            {
              type: 'category',
              label: 'Dev Container',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/container/basics' },
              items: [
                'configuration/dev/container/selector',
                'configuration/dev/container/basics'
              ],
            },
            {
              type: 'category',
              label: 'Ports',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/ports/basics' },
              items: [
                'configuration/dev/ports/forwarding',
                'configuration/dev/ports/reverse-forwarding',
                'configuration/dev/workflow/open',
              ],
            },
            {
              type: 'category',
              label: 'Files',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/files/basics' },
              items: [
                'configuration/dev/files/sync',
                'configuration/dev/files/persist-paths',
                'configuration/dev/files/persistence-options',
              ],
            },
            {
              type: 'category',
              label: 'Dev Workflow',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/workflow/basics' },
              items: [
                'configuration/dev/workflow/terminal',
                'configuration/dev/workflow/log-streaming',
                'configuration/dev/workflow/attach',
                'configuration/dev/workflow/ssh',
                'configuration/dev/workflow/proxy-commands',
                'configuration/dev/container/restarthelper',
              ],
            },
          ],
        },
        'configuration/pullSecrets/basics',
        {
          type: 'category',
          label: 'vars',
          link: { type: 'doc', id: 'configuration/variables/basics' },
          items: [
            'configuration/variables/static',
            'configuration/variables/environment',
            'configuration/variables/command',
            'configuration/variables/question',
            'configuration/variables/env-file',
            'configuration/variables/built-in',
          ],
        },
        'configuration/commands/basics',
        {
          type: 'category',
          label: 'dependencies',
          link: { type: 'doc', id: 'configuration/dependencies/basics' },
          items: [
            'configuration/dependencies/git-repository',
            'configuration/dependencies/local-folder',
          ],
        },
        'configuration/require/basics',
      ],
    },
    {
      type: 'category',
      label: 'devspace --help',
      className: 'code-style',
      link: { type: 'doc', id: 'commands/devspace' },
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
    "plugins/basics",
    {
      type: 'link',
      label: '↗️ Component Chart',
      href: 'https://devspace.sh/component-chart/docs',
    },
  ],
};
