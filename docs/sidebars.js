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
        'configuration/imports/README',
        'configuration/functions/README',
        'configuration/pipelines/README',
        {
          type: 'category',
          label: 'images',
          link: { type: 'doc', id: 'configuration/images/README' },
          items: [
            'configuration/images/build',
            'configuration/images/tag',
            'configuration/images/push',
            'configuration/images/pull',
            /*
            {
              type: 'category',
              label: 'Registry Auth',
              link: { type: 'doc', id: 'configuration/images/registries/README' },
              items: [
                'configuration/images/registries/docker-hub',
                'configuration/images/registries/github',
                'configuration/images/registries/aws',
                'configuration/images/registries/google',
                'configuration/images/registries/azure',
                'configuration/images/registries/other',
              ],
            },*/
            {
              type: 'category',
              label: 'Build Engines',
              link: { type: 'doc', id: 'configuration/images/build-engines/README' },
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
          link: { type: 'doc', id: 'configuration/deployments/README' },
          items: [
            {
              type: 'category',
              label: 'helm',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/deployments/helm/README' },
              items: [
                {
                  type: 'category',
                  label: 'Chart',
                  collapsible: false,
                  link: { type: 'doc', id: 'configuration/deployments/helm/chart/README' },
                  items: [
                    'configuration/deployments/helm/chart/component-chart',
                    'configuration/deployments/helm/chart/local',
                    'configuration/deployments/helm/chart/remote',
                  ],
                },
                'configuration/deployments/helm/values',
              ],
            },
            {
              type: 'category',
              label: 'kubectl',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/deployments/kubectl/README' },
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
          link: { type: 'doc', id: 'configuration/dev/README' },
          items: [
            {
              type: 'category',
              label: 'Dev Container',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/container/README' },
              items: [
                'configuration/dev/container/selector',
                'configuration/dev/container/README'
              ],
            },
            {
              type: 'category',
              label: 'Ports',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/ports/README' },
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
              link: { type: 'doc', id: 'configuration/dev/files/README' },
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
              link: { type: 'doc', id: 'configuration/dev/workflow/README' },
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
        'configuration/pullSecrets/README',
        {
          type: 'category',
          label: 'vars',
          link: { type: 'doc', id: 'configuration/variables/README' },
          items: [
            'configuration/variables/static',
            'configuration/variables/environment',
            'configuration/variables/command',
            'configuration/variables/question',
            'configuration/variables/env-file',
            'configuration/variables/built-in',
          ],
        },
        'configuration/commands/README',
        {
          type: 'category',
          label: 'dependencies',
          link: { type: 'doc', id: 'configuration/dependencies/README' },
          items: [
            'configuration/dependencies/git-repository',
            'configuration/dependencies/local-folder',
          ],
        },
        'configuration/require/README',
      ],
    },
    {
      type: 'category',
      label: 'devspace --help',
      className: 'code-style',
      link: { type: 'doc', id: 'commands/devspace' },
      items: [
        {
          type: 'autogenerated',
          dirName: 'commands',
          className: 'code-style',
        },
      ],
    },
    "plugins/README",
    {
      type: 'link',
      label: '↗️ Component Chart',
      href: 'https://devspace.sh/component-chart/docs',
    },
  ],
};
