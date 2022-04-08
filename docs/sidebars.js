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
      items: [
        {
          type: 'category',
          label: 'pipelines',
          link: { type: 'doc', id: 'configuration/pipelines/basics' },
          items: [
            {
              type: 'category',
              label: 'Images',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/build_images',
                'configuration/pipelines/functions/ensure_pull_secrets',
                'configuration/pipelines/functions/get_image',
              ],
            },
            {
              type: 'category',
              label: 'Deployments',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/create_deployments',
                'configuration/pipelines/functions/purge_deployments',
              ],
            },
            {
              type: 'category',
              label: 'Development',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/start_dev',
                'configuration/pipelines/functions/stop_dev',
              ],
            },
            {
              type: 'category',
              label: 'Pipelines',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/run_pipelines',
                'configuration/pipelines/functions/run_default_pipeline',
                'configuration/pipelines/functions/run_dependency_pipelines',
              ],
            },
            {
              type: 'category',
              label: 'Checks',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/is_command',
                'configuration/pipelines/functions/is_dependency',
                'configuration/pipelines/functions/is_equal',
                'configuration/pipelines/functions/is_os',
                'configuration/pipelines/functions/is_true',
              ],
            },
            {
              type: 'category',
              label: 'Other',
              collapsible: false,
              items: [
                'configuration/pipelines/functions/cat',
                'configuration/pipelines/functions/exec_container',
                'configuration/pipelines/functions/get_config_value',
                'configuration/pipelines/functions/run_watch',
                'configuration/pipelines/functions/select_pod',
                'configuration/pipelines/functions/sleep',
                'configuration/pipelines/functions/xargs',
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
                'configuration/dev/container/devimage',
                'configuration/dev/container/env',
                'configuration/dev/container/workingdir',
                'configuration/dev/container/command-args',
                'configuration/dev/container/resources',
                'configuration/dev/container/patches',
                'configuration/dev/container/restarthelper',
                'configuration/dev/container/arch',
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
              label: 'Ports',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/ports/basics' },
              items: [
                'configuration/dev/ports/forwarding',
                'configuration/dev/ports/reverse-forwarding',
              ],
            },
            {
              type: 'category',
              label: 'Dev Workflow',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/workflow/basics' },
              items: [
                'configuration/dev/workflow/attach',
                'configuration/dev/workflow/log-streaming',
                'configuration/dev/workflow/ssh',
                'configuration/dev/workflow/terminal',
                'configuration/dev/workflow/proxy-commands',
                'configuration/dev/workflow/open',
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
          label: 'images',
          link: { type: 'doc', id: 'configuration/images/basics' },
          items: [
            {
              type: 'category',
              label: 'Image Definition',
              collapsible: false,
              items: [
                'configuration/images/image-definition/image',
                'configuration/images/image-definition/tags',
                'configuration/images/image-definition/context',
                'configuration/images/image-definition/dockerfile',
                'configuration/images/image-definition/append-dockerfile-instructions',
                //'configuration/images/entrypoint-cmd',
              ],
            },
            {
              type: 'category',
              label: 'Build Settings',
              collapsible: false,
              items: [
                'configuration/images/build-settings/build-args',
                'configuration/images/build-settings/target',
                'configuration/images/build-settings/network',
                'configuration/images/build-settings/rebuild-strategy',
                //'configuration/images/inject-restart-helper',
                //'configuration/images/pull-secrets',
                //'configuration/images/skip-push',
              ],
            },
            {
              type: 'category',
              label: 'Build Engines',
              collapsible: false,
              items: [
                'configuration/images/build-engines/docker',
                'configuration/images/build-engines/buildkit',
                'configuration/images/build-engines/kaniko',
                'configuration/images/build-engines/custom',
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
        {
          type: 'category',
          label: 'commands',
          link: { type: 'doc', id: 'configuration/commands/basics' },
          items: [
            {
              type: 'category',
              label: 'Checks',
              collapsible: false,
              items: [
                'configuration/commands/functions/is_command',
                'configuration/commands/functions/is_equal',
                'configuration/commands/functions/is_os',
                'configuration/commands/functions/is_true',
              ],
            },
            {
              type: 'category',
              label: 'Other',
              collapsible: false,
              items: [
                'configuration/commands/functions/cat',
                'configuration/commands/functions/run_watch',
                'configuration/commands/functions/sleep',
                'configuration/commands/functions/xargs',
              ],
            },
          ],
        },
        'configuration/functions/basics',
        'configuration/imports/basics',
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
