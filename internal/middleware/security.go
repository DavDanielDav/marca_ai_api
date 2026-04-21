package middleware

import (
	"net/http"
	"path"
	"strings"
)

func DenySensitivePathsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSensitiveRequestPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSensitiveRequestPath(requestPath string) bool {
	cleanPath := strings.TrimSpace(requestPath)
	if cleanPath == "" {
		return false
	}

	cleanPath = path.Clean("/" + cleanPath)
	segments := strings.Split(cleanPath, "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		lowerSegment := strings.ToLower(segment)
		if lowerSegment == ".well-known" {
			continue
		}

		if strings.HasPrefix(lowerSegment, ".") {
			return true
		}
	}

	return false
}
