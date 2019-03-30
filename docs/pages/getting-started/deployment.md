---
title: 2. Deploy with DevSpace
---

You are now ready to deploy your application to a kubernetes cluster. With DevSpace you can create a so called [Space](/docs/cloud/spaces/what-are-spaces) that is basically an isolated hosted kubernetes namespace. You can also just use any other kubernetes cluster.


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


## Optional: Create a Space

With the following command, you can create a Space called `myapp`:
```bash
devspace create space myapp
```

DevSpace CLI now will automatically work with the Space that you just created.

## Deploy your application
Now, you can deploy your application with the following command:
```bash
devspace deploy
```

This command will do the following:
1. Build a [Docker image](/docs/cli/deployment/images) as defined in the `Dockerfile`
2. Push this Docker image to any [Docker registry](/docs/cli/images/workflow) 
3. Deploy your [Helm chart](/docs/chart/basics/what-are-helm-charts) as defined in `chart/`
4. If you are using a space: Make your application available on a `.devspace.host` domain

You should receive an output similar to this:
```bash
[info]   Loaded config from .devspace/configs.yaml
[info]   Building image 'registry.devspace.rocks/myuser/devspace' with engine 'docker'
[done] √ Authentication successful (registry.devspace.rocks)
Sending build context to Docker daemon  283.6kB
Step 1/9 : FROM node:8.11.4
 ---> 8198006b2b57
[...]
hKEA2Kr: digest: sha256:ae6e096757da670907c41935646c4a87a5118801947af150052f5eccf4ed226d size: 2841
[info]   Image pushed to registry (registry.devspace.rocks)
[done] √ Done processing image 'registry.devspace.rocks/myuser/devspace'
[info]   Deploying devspace-app with helm
[done] √ Deployed helm chart (Release revision: 3)                                            
[done] √ Finished deploying devspace-app
[done] √ Successfully deployed!
```

Congrats you have successfully deployed an application to kubernetes!

## Learn more about deploying with DevSpace
See the following guides to learn more:
- [Develop with DevSpace](/docs/getting-started/development)
- [Connect custom domains](/docs/cli/deployment/domains)
- [Monitor and debug deployed applications](/docs/cli/debugging/overview)
- [Configure Docker image](/docs/cli/deployment/images)
- [Configure Helm chart](/docs/cli/deployment/charts)
- [What are components?](/docs/chart/basics/components)
- [Configure persistent volumes](/docs/chart/customization/persistent-volumes)
- [Configure environment variables](/docs/chart/customization/environment-variables)
- [Configure networking for your Helm chart (e.g. ingress)](/docs/chart/customization/networking)
- [Add a database](/docs/chart/customization/predefined-components)
- [Add a custom component](/docs/chart/customization/add-component)
- [Add a container](/docs/chart/customization/containers)
- [Add custom Kubernetes manifests (.yaml files)](/docs/chart/customization/custom-manifests)
