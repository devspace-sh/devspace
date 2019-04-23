package walk

import "testing"

func TestWalk(t *testing.T) {
	// @Florian
	// Take a look at how the walk function works and transform the below input into the output below

	// Input yaml
	_ = `
test: 123
	image: appendtag
	test: []
test2:
	image: dontreplaceme
	test3:
	- test4:
		test5:
		image: replaceme
	`

	// Output yaml
	_ = `
test: 123
	image: appendtag:test
	test: []
test2:
	image: dontreplaceme
	test3:
	- test4:
		test5:
		image: replaced
		`
}
