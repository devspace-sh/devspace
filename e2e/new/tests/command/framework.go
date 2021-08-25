package command

import "github.com/onsi/ginkgo"

// DevSpaceDescribe annotates the test with the label.
func DevSpaceDescribe(text string, body func()) bool {
	return ginkgo.Describe("[command] "+text, body)
}
