package message

// ConfigNotFound should be used if devspace.yaml cannot be found
const ConfigNotFound = "Cannot find a devspace.yaml for this project. Please run `devspace init`"

// ServiceNotFound should be used if there are no Kubernetes services resources
const ServiceNotFound = "Cannot find any services in namespace '%s'. Please make sure you have a service that this ingress can connect to. \n\nTo get a list of services in your current namespace, run: kubectl get services"
