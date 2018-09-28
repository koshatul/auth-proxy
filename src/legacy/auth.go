package legacy

import (
	"crypto/subtle"
	"net/http"

	"github.com/koshatul/auth-proxy/src/httpauth"
	"go.uber.org/zap"
)

// AuthItem is a single authentication item for use with legacy auth.
type AuthItem struct {
	Username string
	Password string
}

// AuthCheckFunc returns a authentication check function for use with `httpauth.BasicAuth()``
func AuthCheckFunc(logger *zap.Logger, legacyAuthItems map[string]AuthItem, authProvider httpauth.AuthProvider) httpauth.AuthProvider {
	return func(username, password string, r *http.Request) (string, bool) {
		if len(legacyAuthItems) > 0 {
			// Do Legacy Auth
			if v, ok := legacyAuthItems[username]; ok {
				if subtle.ConstantTimeCompare([]byte(password), []byte(v.Password)) == 1 {
					logger.Debug("Auth Success[legacy]", zap.String("username", username))
					return v.Username, true
				} else {
					logger.Info("Auth Failure[legacy]", zap.String("username", username))
					return "", false
				}
			}
			return authProvider(username, password, r)
		} else {
			return authProvider(username, password, r)
		}
	}
}
