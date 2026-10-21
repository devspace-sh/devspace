package commands

import "testing"

func TestMatchesPatterns(t *testing.T) {
	patterns := []watchPattern{
		{base: "/project", pattern: "devspace.yaml"},
		{base: "/var/tmp/chart", pattern: "**"},
	}

	testCases := []struct {
		name  string
		event watchEvent
		want  bool
	}{
		{"exact file in its own base", watchEvent{base: "/project", path: "devspace.yaml"}, true},
		{"unrelated file in project base", watchEvent{base: "/project", path: "main.go"}, false},
		{"nested unrelated file in project base", watchEvent{base: "/project", path: "src/main.go"}, false},
		{"any file in chart base", watchEvent{base: "/var/tmp/chart", path: "templates/deploy.yaml"}, true},
		{"unknown base", watchEvent{base: "/other", path: "devspace.yaml"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesPatterns(patterns, tc.event); got != tc.want {
				t.Fatalf("matchesPatterns(%v) = %v, want %v", tc.event, got, tc.want)
			}
		})
	}
}
