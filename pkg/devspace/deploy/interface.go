package deploy

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Delete(silent bool) error
}
