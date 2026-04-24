package util

import "testing"

func TestHeadingPrefixCapsAtLevelSix(t *testing.T) {
	testCases := []struct {
		name     string
		depth    int
		expected string
	}{
		{name: "top level", depth: 1, expected: "## "},
		{name: "sixth level", depth: 5, expected: "###### "},
		{name: "deeper than six", depth: 8, expected: "###### "},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := headingPrefix(testCase.depth); actual != testCase.expected {
				t.Fatalf("headingPrefix(%d) = %q, want %q", testCase.depth, actual, testCase.expected)
			}
		})
	}
}
