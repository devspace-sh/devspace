---
title: 4. Why use DevSpace during development?
---

# Why use DevSpace for development
The basic idea behind DevSpace is to bring the different software execution environments during development, staging, testing and production closer together. One issue with current cloud-native application development is that the development environment where code is actually written often greatly differs from the environment where it is deployed to. In bigger software teams, development execution environments often even differ from each other, which can lead to numerous problems and inefficiencies during development and the eventual execution.  

One solution to this problem is to use kubernetes not only during the later stages of the software development process, but instead from the beginning on. This has the advantage that the developer can continuously test during development how the application will run, scale and behave in a kubernetes environment. Furthermore the execution environment can be simply shared with other developers in the team through the kubernetes .yaml files. We believe that this approach would lead to ultimately better cloud-native applications.  

However, while this idea sounds tempting, using kubernetes during development currently is a real pain for a developer. Kubernetes clusters are not easy to configure, many features require deep knowledge of kubernetes and it is not easy to develop code in remote containers without redeploying after every change. The ambition of DevSpace is to facilitate and speed up development with kubernetes and keep the actual workflow of the developer as close as possible to the previous local development workflow.  

# How is it different to other kubernetes development tools
There are already several other tools that facilitate the continuous development of kubernetes applications. While these tools are a good beginning, they unfortunately in our opinion don't provide a real local coding experience and have several drawbacks.  
  
## draft & skaffold
Draft and skaffold facilitate continuous development for kubernetes applications. They support the workflow for building, pushing and deploying the application. While they automate and abstract some complexity during building and deploying applications, the problem with these tools is in our opinion that continuous development is tedious, because after each code change a new pipeline with image building and deploying is started. Our approach let's the developer exchange the files that changed directly in the target container without the need to redeploy after every change.  

## telepresence
Telepresence substitutes a two-way network proxy for your normal pod running in the Kubernetes cluster. This pod proxies data from your Kubernetes environment to the local process. The local process has its networking transparently overridden so that DNS calls and TCP connections are routed through the proxy to the remote Kubernetes cluster. While this sounds great at first, the approach has several drawbacks: accessing localhost in the container refers to the local machine and not the pod ip, volumes only work with code changes and there is no windows support.

## ksync
