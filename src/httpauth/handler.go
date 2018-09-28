package httpauth

import (
	"net/http"
)

// BasicAuthHandler needs a comment
type BasicAuthHandler struct {
	Handler    http.Handler
	RemoveAuth bool

	*BasicAuthWrapper
}

// Satisfies the http.Handler interface for basicAuth.
func (b *BasicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if we have a user-provided error handler, else set a default
	if b.UnauthorizedHandler == nil {
		b.UnauthorizedHandler = http.HandlerFunc(defaultUnauthorizedHandler)
	}

	var username string
	var ok bool
	// Check that the provided details match
	if username, ok = b.authenticate(r); !ok {
		b.requestAuth(w, r)
		return
	}
	if b.RemoveAuth {
		r.Header.Set("X-Username", username)
		r.Header.Del("Authorization")
	}

	// Call the next handler on success.
	b.Handler.ServeHTTP(w, r)
}
