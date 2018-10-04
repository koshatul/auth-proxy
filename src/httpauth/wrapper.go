package httpauth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	cache "github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AuthProvider is a function that given a username, password and request, authenticates the user.
type AuthProvider func(username string, password string, r *http.Request) (string, bool)

// AuthenticatedRequest replaces Request in Request handlers instead of http.Request
type AuthenticatedRequest struct {
	http.Request
	/*
	 Authenticated user name. Current API implies that Username is
	 never empty, which means that authentication is always done
	 before calling the request handler.
	*/
	Username string
}

// AuthenticatedHandlerFunc is like http.HandlerFunc, but takes
// AuthenticatedRequest instead of http.Request
type AuthenticatedHandlerFunc func(http.ResponseWriter, *AuthenticatedRequest)

// BasicAuthWrapper needs a comment
type BasicAuthWrapper struct {
	Cache               *cache.Cache
	Realm               string
	Logger              *zap.Logger
	AuthFunc            AuthProvider
	UnauthorizedHandler http.Handler
}

/*
Wrap BasicAuthenticator returns a function, which wraps an
AuthenticatedHandlerFunc converting it to http.HandlerFunc. This
wrapper function checks the authentication and either sends back
required authentication headers, or calls the wrapped function with
authenticated username in the AuthenticatedRequest.
*/
func (b *BasicAuthWrapper) Wrap(wrapped AuthenticatedHandlerFunc) http.HandlerFunc {
	if b.UnauthorizedHandler == nil {
		b.UnauthorizedHandler = http.HandlerFunc(defaultUnauthorizedHandler)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Check that the provided details match
		if username, ok := b.authenticate(r); ok {
			ar := &AuthenticatedRequest{Request: *r, Username: username}
			wrapped(w, ar)
			return
		}
		b.requestAuth(w, r)
	}
}

// Require authentication, and serve our error handler otherwise.
func (b *BasicAuthWrapper) requestAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, b.Realm))
	// w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")

	b.UnauthorizedHandler.ServeHTTP(w, r)
}

type cachedResponse struct {
	Username string
	Result   bool
}

// authenticate retrieves and then validates the user:password combination provided in
// the request header. Returns 'false' if the user has not successfully authenticated.
func (b *BasicAuthWrapper) authenticate(r *http.Request) (string, bool) {
	const basicScheme string = "Basic "

	if r == nil {
		return "", false
	}

	// If AuthFunc is missing, fail logins
	if b.AuthFunc == nil {
		return "", false
	}

	if v, ok := b.Cache.Get(r.Header.Get("Authorization")); ok {
		// ACL Record cached
		resp := v.(cachedResponse)
		if resp.Result {
			r.URL.User = url.User(resp.Username)
		}
		return resp.Username, resp.Result
	}

	givenUser, givenPass, err := GetBasicAuthFromRequest(r)
	if err != nil {
		return "", false
	}

	authUser, authResult := b.AuthFunc(givenUser, givenPass, r)
	b.Cache.Set(
		r.Header.Get("Authorization"),
		cachedResponse{
			Username: authUser,
			Result:   authResult,
		},
		viper.GetDuration("server.cache.default-expire"),
	)
	if authResult {
		r.URL.User = url.User(authUser)
	}
	return authUser, authResult
}

// defaultUnauthorizedHandler provides a default HTTP 401 Unauthorized response.
func defaultUnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

// GetBasicAuthFromRequest returns basic auth username and password given a `*http.Request`
func GetBasicAuthFromRequest(r *http.Request) (string, string, error) {
	const basicScheme string = "Basic "

	if r == nil {
		return "", "", fmt.Errorf("request is nil")
	}

	// Confirm the request is sending Basic Authentication credentials.
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, basicScheme) {
		return "", "", fmt.Errorf("basic auth headers missing")
	}

	// Get the plain-text username and password from the request.
	// The first six characters are skipped - e.g. "Basic ".
	str, err := base64.StdEncoding.DecodeString(auth[len(basicScheme):])
	if err != nil {
		return "", "", err
	}

	// Split on the first ":" character only, with any subsequent colons assumed to be part
	// of the password. Note that the RFC2617 standard does not place any limitations on
	// allowable characters in the password.
	creds := bytes.SplitN(str, []byte(":"), 2)

	if len(creds) != 2 {
		return "", "", fmt.Errorf("basic auth format invalid")
	}

	return string(creds[0]), string(creds[1]), nil
}
