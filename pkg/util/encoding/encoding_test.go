package encoding

import (
	"gotest.tools/assert"
	"testing"
)

type testCase struct {
	name string
	safe bool
}

func TestIsUnsafeUpperName(t *testing.T) {
	cases := []testCase{
		{
			name: "a",
			safe: true,
		},
		{
			name: "a--b",
			safe: true,
		},
		{
			name: "a- -b",
			safe: false,
		},
		{
			name: "a-$-b",
			safe: false,
		},
		{
			name: "a-_-b",
			safe: false,
		},
		{
			name: "-ab",
			safe: false,
		},
		{
			name: "AB",
			safe: false,
		},
	}
	upperCases := []testCase{
		{
			name: "a",
			safe: true,
		},
		{
			name: "a-_B",
			safe: true,
		},
		{
			name: "a--b",
			safe: true,
		},
		{
			name: "A__B",
			safe: true,
		},
		{
			name: "A_ _B",
			safe: false,
		},
		{
			name: "A_%_B",
			safe: false,
		},
		{
			name: "A_%$",
			safe: false,
		},
		{
			name: "-ABV",
			safe: false,
		},
		{
			name: "ABV_",
			safe: false,
		},
	}

	for _, c := range cases {
		unsafe := IsUnsafeName(c.name)
		assert.Equal(t, !unsafe, c.safe, "expect "+c.name)
	}
	for _, c := range upperCases {
		unsafe := IsUnsafeUpperName(c.name)
		assert.Equal(t, !unsafe, c.safe, "expect upper "+c.name)
	}
}
