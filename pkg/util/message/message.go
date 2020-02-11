package message

// ConfigNotFound should be used if devspace.yaml cannot be found
const ConfigNotFound = "Cannot find a devspace.yaml for this project. Please run `devspace init`"

// ConfigNoImages should be used if devpsace.yaml does not contain any images
const ConfigNoImages = "This project's devspace.yaml does not contain any images. Interactive mode requires at least one image because it overrides the entrypoint of one of the specified images. \n\nAlternative commands: \n- `devspace enter` to open a terminal (without port-forwarding and file sync) \n- `devspace dev -t` to start development mode (port-forwarding and file sync) but open the terminal instead of streaming the logs"

// SpaceNotFound should be used if the Space %s does not exist
const SpaceNotFound = "Cannot find Space '%s' \n\nYou may need to run `devspace login` or prefix the Space name with the name of its owner, e.g. USER:SPACE"

// SpaceQueryError should be used if a graphql query for a space fails
const SpaceQueryError = "Error retrieving Space details"

// SelectorErrorPod should be used if there is an error selecting a pod
const SelectorErrorPod = "Error selecting pod"

// SelectorLabelNotFound should be used when selecting pod via labelSelector finds no pods
const SelectorLabelNotFound = "Cannot find a pod using label selector '%s' in namespace '%s' \n\nTo get a list of all pods with their labels, run: kubectl get pods --show-labels"

// PodStatusCritical should be used if a pod has a critial status but is expected to be running (or completed)
const PodStatusCritical = "Pod '%s' has critical status '%s' \n\nTo get more information about this pod's status, run: kubectl describe pod %s"

// ServiceNotFound should be used if there are no Kubernetes services resources
const ServiceNotFound = "Cannot find any services in namespace '%s'. Please make sure you have a service that this ingress can connect to. \n\nTo get a list of services in your current namespace, run: kubectl get services"
