package sync

import "github.com/onsi/ginkgo/v2"

// DevSpaceDescribe annotates the test with the label.
func DevSpaceDescribe(text string, body func()) bool {
	return ginkgo.Describe("[sync] "+text, body)
}
