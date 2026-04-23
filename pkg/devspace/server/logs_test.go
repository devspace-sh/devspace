package server

import (
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestCheckOrigin(t *testing.T) {
	cases := map[string]struct {
		origin string
		want   bool
	}{
		"no origin header (CLI/curl)": {origin: "", want: true},
		"localhost origin":            {origin: "http://localhost:8080", want: true},
		"127.0.0.1 origin":            {origin: "http://127.0.0.1:3000", want: true},
		"localhost no port":           {origin: "http://localhost", want: true},
		"external origin":             {origin: "http://bad.example.com", want: false},
		"invalid origin":              {origin: "://bad-url", want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			got := upgrader.CheckOrigin(r)
			assert.Equal(t, tc.want, got, name)
		})
	}
}
