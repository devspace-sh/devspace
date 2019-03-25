---
title: 2. Containerize your app
---

If you want to deploy your application with DevSpace CLI, your application will need a working Dockerfile. DevSpace can also [create a dockerfile for your project](/docs/cli/deployment/containerize-your-app), if you don't have one. 

If you do not have a project to work with, you can **checkout one of our demo projects (optional)**
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```json
git clone https://github.com/devspace-cloud/quickstart-nodejs
cd quickstart-nodejs
```

<!--Python-->
```json
git clone https://github.com/devspace-cloud/quickstart-python
cd quickstart-python
```

<!--Golang-->
```json
git clone https://github.com/devspace-cloud/quickstart-golang
cd quickstart-golang
```

<!--Ruby-->
```json
git clone https://github.com/devspace-cloud/quickstart-ruby
cd quickstart-ruby
```

<!--Php-->
```json
git clone https://github.com/devspace-cloud/quickstart-php
cd quickstart-php
```

<!--END_DOCUSAURUS_CODE_TABS-->

> You can also use any existing project with Dockerfile. If you do not have a Dockerfile take a look at [containerize an existing project](/docs/cli/deployment/containerize-your-app). DevSpace works with every Dockerfile.

## Initialize your project
Run the following command within your project to initialize DevSpace:
```bash
devspace init
```

DevSpace CLI will automatically create the following files:
```bash
project/                    # your project directory
|
|--.devspace/               # DevSpace directory
|   |-config.yaml           # DevSpace config
|
|--chart/                   # Helm chart (defines how to deploy your application)
|   |-Chart.yaml            # chart definition (e.g. name, version)
|   |-values.yaml           # values for the template variables
|   |-templates/            # directory containing the template files
```

<details>
<summary>
### Learn how to customize Helm chart and image building (optional)
</summary>

See the following guides to:
- [Configure image building](/docs/cli/deployment/images)
- [What are components?](/docs/chart/basics/components)
- [Configure persistent volumes](/docs/chart/customization/persistent-volumes)
- [Configure environment variables](/docs/chart/customization/environment-variables)
- [Configure networking for your Helm chart (e.g. ingress)](/docs/chart/customization/networking)
- [Add a database](/docs/chart/customization/predefined-components)
- [Add a custom component](/docs/chart/customization/add-component)
- [Add a container](/docs/chart/customization/containers)
- [Add custom Kubernetes manifests (.yaml files)](/docs/chart/customization/custom-manifests)

</details>
