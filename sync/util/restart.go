package util

type ContainerRestarter interface {
	RestartContainer() error
}
