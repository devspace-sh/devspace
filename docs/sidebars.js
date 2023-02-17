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
        'getting-started/cleanup',
       // 'getting-started/next-steps',
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
        'configuration/hooks/README',
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
                'configuration/deployments/helm/values',
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
              ],
            },
            {
              type: 'category',
              label: 'kubectl',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/deployments/kubectl/README' },
              items: [
                {
                  type: 'category',
                  label: 'Manifests',
                  collapsible: false,
                  link: { type: 'doc', id: 'configuration/deployments/kubectl/README' },
                  items: [
                    'configuration/deployments/kubectl/manifests',
                    'configuration/deployments/kubectl/inline_manifests',
                    'configuration/deployments/kubectl/patches',
                  ],
                },
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
              label: '1. Select Dev Container',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/selectors/README' },
              className: "extra-indent",
              items: [
                'configuration/dev/selectors/image',
                'configuration/dev/selectors/labels',
              ],
            },
            {
              type: 'category',
              label: '2. Add Dev Connections',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/connections/README' },
              className: "extra-indent",
              items: [
                'configuration/dev/connections/file-sync',
                'configuration/dev/connections/port-forwarding',
                'configuration/dev/connections/terminal',
                'configuration/dev/connections/ssh',
                'configuration/dev/connections/restart-helper',
                'configuration/dev/connections/proxy-commands',
                'configuration/dev/connections/open',
              ],
            },
            {
              type: 'category',
              label: '3. Modify Dev Container',
              collapsible: false,
              link: { type: 'doc', id: 'configuration/dev/modifications/README' },
              className: "extra-indent",
              items: [
                'configuration/dev/modifications/dev-image',
                'configuration/dev/modifications/env-vars',
                'configuration/dev/modifications/entrypoint',
                'configuration/dev/modifications/workdir',
                'configuration/dev/modifications/persistence',
                'configuration/dev/modifications/resources',
                'configuration/dev/modifications/patches',
              ],
            },
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
        {
          type: 'category',
          label: 'profiles',
          link: { type: 'doc', id: 'configuration/profiles/README' },
          items: [
            'configuration/profiles/activation',
            'configuration/profiles/patches',
            'configuration/profiles/merge',
            'configuration/profiles/replace',
          ],
        },
        'configuration/pullSecrets/README',
        'configuration/localRegistry/README',
        'configuration/require/README',
        'configuration/variables',
        'configuration/runtime-variables',
        'configuration/expressions',
      ],
    },
    {
      type: 'category',
      label: 'IDE Integration',
      link: { type: 'doc', id: 'ide-integration/visual-studio-code' },
      items: [
        'ide-integration/visual-studio-code',
      ],
    },
    {
      type: 'category',
      label: 'devspace --help',
      className: 'code-style',
      link: { type: 'doc', id: 'cli' },
      items: [
        {
          type: 'autogenerated',
          dirName: 'cli',
          className: 'code-style',
        },
      ],
    },
    "plugins/README",
  ],
};
