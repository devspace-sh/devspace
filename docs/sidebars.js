/**
 * Copyright (c) 2017-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

module.exports = {
  adminSidebar: [
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
      label: 'devspace.yaml',
      className: 'code-style',
      link: { type: 'doc', id: 'configuration/reference' },
      items: [
        {
          type: 'category',
          label: 'pipelines',
          link: { type: 'doc', id: 'configuration/pipelines/basics' },
          items: [
            'configuration/pipelines/functions/build_images',
            'configuration/pipelines/functions/cat',
            'configuration/pipelines/functions/ensure_pull_secrets',
            'configuration/pipelines/functions/create_deployments',
            'configuration/pipelines/functions/purge_deployments',
            'configuration/pipelines/functions/exec_container',
            'configuration/pipelines/functions/get_config_value',
            'configuration/pipelines/functions/get_image',
            'configuration/pipelines/functions/is_command',
            'configuration/pipelines/functions/is_dependency',
            'configuration/pipelines/functions/is_equal',
            'configuration/pipelines/functions/is_os',
            'configuration/pipelines/functions/is_true',
            'configuration/pipelines/functions/run_default_pipeline',
            'configuration/pipelines/functions/run_dependency_pipelines',
            'configuration/pipelines/functions/run_pipelines',
            'configuration/pipelines/functions/run_watch',
            'configuration/pipelines/functions/select_pod',
            'configuration/pipelines/functions/sleep',
            'configuration/pipelines/functions/start_dev',
            'configuration/pipelines/functions/stop_dev',
            'configuration/pipelines/functions/xargs',
          ],
        },
        {
          type: 'category',
          label: 'dev',
          link: { type: 'doc', id: 'configuration/development/basics' },
          items: [
            'configuration/development/selector',
            'configuration/development/port-forwarding',
            'configuration/development/reverse-port-forwarding',
            'configuration/development/open-links',
            'configuration/development/file-synchronization',
            'configuration/development/dev-image',
            'configuration/development/command-args',
            'configuration/development/workingDir',
            'configuration/development/env-resources',
            'configuration/development/patches',
            'configuration/development/ssh',
            'configuration/development/proxy-commands',
            'configuration/development/attach',
            'configuration/development/terminal',
            'configuration/development/log-streaming',
          ],
        },
        {
          type: 'category',
          label: 'deployments',
          link: { type: 'doc', id: 'configuration/deployments/basics' },
          items: [
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
          label: 'images',
          link: { type: 'doc', id: 'configuration/images/basics' },
          items: [
            'configuration/images/image-tagging',
            'configuration/images/dockerfile-context',
            'configuration/images/entrypoint-cmd',
            'configuration/images/append-dockerfile-instructions',
            'configuration/images/inject-restart-helper',
            'configuration/images/rebuild-strategy',
            'configuration/images/pull-secrets',
            'configuration/images/build-args',
            'configuration/images/network',
            'configuration/images/skip-push',
            'configuration/images/target',
            'configuration/images/docker',
            'configuration/images/buildkit',
            'configuration/images/kaniko',
            'configuration/images/custom',
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
            'configuration/commands/functions/cat',
            'configuration/commands/functions/is_command',
            'configuration/commands/functions/is_equal',
            'configuration/commands/functions/is_os',
            'configuration/commands/functions/is_true',
            'configuration/commands/functions/run_watch',
            'configuration/commands/functions/sleep',
            'configuration/commands/functions/xargs',
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
    {
      type: 'link',
      label: '↗️ Component Chart',
      href: 'https://devspace.sh/component-chart/docs',
    },
  ],
};
