package httputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalhostURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://localhost:8080/path", true},
		{"https://localhost/pack", true},
		{"http://127.0.0.1:9090/test", true},
		{"http://[::1]:8080/path", true},
		{"https://www.swi-prolog.org/pack/list", false},
		{"https://github.com/repo", false},
		{"://malformed", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.want, IsLocalhostURL(tt.url))
		})
	}
}
