package browser

import "github.com/skratchdot/open-golang/open"

// Browser can open the default browser
type Browser interface {
	Run(url string) error
	Start(url string) error
}

type browser struct{}

// NewBrowser creates an instance of the interface Browser
func NewBrowser() Browser {
	return &browser{}
}

func (b *browser) Run(url string) error {
	return open.Run(url)
}

func (b *browser) Start(url string) error {
	return open.Start(url)
}
