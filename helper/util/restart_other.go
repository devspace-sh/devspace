//go:build !linux
// +build !linux

package util

func NewContainerRestarter() ContainerRestarter {
	return nil
}
