---
title: 2. Containerize your app
---

DevSpace.cli lets you easily containerize your application, so you can deploy it to Kubernetes. You can use one of your own projects for the remainder of this guide. 

If you do not have a project to work with, you can **checkout one of our demo projects (optional)**
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```bash
git clone https://github.com/devspace-cloud/devspace-quickstart-nodejs
cd devspace-quickstart-nodejs
```

<!--END_DOCUSAURUS_CODE_TABS-->

> You can also use any existing project. DevSpace works with any programming language.

## Initialize your project
Run the following command within your project:
```bash
devspace init
```
DevSpace.cli will automatically detect your programming language and ask for the ports your application is listening on. Then, it will automatically create the following files:
```bash
project/
|
|--.devspace/
|   |
|   |-config.yaml
|
|--chart/
|--Dockerfile
```

> You can fully customize the deployment configuration created by `devspace init`.

<!-- >
<details>
<summary>
### **Show customization options**

</summary>

#### Customize the Dockerfile

#### Customize the Helm chart

#### Advanced Customizations

</details>
<!-- -->
