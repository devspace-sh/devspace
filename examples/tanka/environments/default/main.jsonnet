local app = (import 'app.libsonnet');

{
  apiVersion: 'tanka.dev/v1alpha1',
  kind: 'Environment',
  metadata: {
    name: std.extVar('DEVSPACE_NAME'),
  },
  spec: {
    // Use inline configuration so DevSpace can pass namespace/apiserver to tanka
    apiServer: std.extVar('DEVSPACE_API_SERVER'),

    namespace: std.extVar('DEVSPACE_NAMESPACE'),
    injectLabels: true,
  },
  data: app,
}
