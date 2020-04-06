// +build !linux

package util

type containerRestarter struct {
}

func NewContainerRestarter() ContainerRestarter {
	return nil
}

func (*containerRestarter) RestartContainer() error {
	panic("not implemented")
}

