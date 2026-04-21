package middleware

import "testing"

func TestIsSensitiveRequestPath(t *testing.T) {
	tests := []struct {
		path      string
		sensitive bool
	}{
		{path: "/", sensitive: false},
		{path: "/health", sensitive: false},
		{path: "/.env", sensitive: true},
		{path: "/api/.git/config", sensitive: true},
		{path: "/.gitignore", sensitive: true},
		{path: "/.env.production", sensitive: true},
		{path: "/.well-known/acme-challenge/token", sensitive: false},
	}

	for _, test := range tests {
		if got := isSensitiveRequestPath(test.path); got != test.sensitive {
			t.Fatalf("path %q: expected sensitive=%v, got %v", test.path, test.sensitive, got)
		}
	}
}
