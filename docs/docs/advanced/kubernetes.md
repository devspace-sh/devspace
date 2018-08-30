---
title: Kubernetes Guides
---

To create DevSpaces, you need a Kubernetes cluster. The following guides provide some help on setting up a Kubernetes cluster.

## Minikube
1. [Install Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
2. Start Minikube:
    - **Option A**: Running minikube with VirtualBox
        ```
        minikube start --vm-driver="virtualbox" -v 9999
        ```
    - **Option B**: Running minikube with Hyper-V (Windows only) using **powershell**
        ```powershell
        Get-NetAdapter # COPY THE NAME OF YOUR DETAILS NETWORK ADAPTER
        New-VMSwitch -Name "minikube" -NetAdapterName "DEFAULT_NETWORK_ADAPTER" -AllowManagementOS $true
        minikube start --vm-driver="hyperv" --hyperv-virtual-switch="minikube" -v 9999
        ```

Additional information can be found in the official documentation: **[Minikube Quickstart](https://kubernetes.io/docs/setup/minikube/#quickstart)**
