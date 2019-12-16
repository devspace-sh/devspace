package testing

// FakeBrowser is a fake browser implementation for test purposes
type FakeBrowser struct {
	RunCallback   callback
	StartCallback callback
}

type callback func(url string) error

// Run is a fake implementation. It calls RunCallback
func (b *FakeBrowser) Run(url string) error {
	return b.RunCallback(url)
}

// Start is a fake implementation. It calls StartCallback
func (b *FakeBrowser) Start(url string) error {
	return b.StartCallback(url)
}
