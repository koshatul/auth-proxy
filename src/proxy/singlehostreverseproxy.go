package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewSingleHostReverseProxy returns a new ReverseProxy that routes
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
// NewSingleHostReverseProxy rewrites the Host header, and uses a custom tlsConfig.
func NewSingleHostReverseProxy(target *url.URL, passHostHeader bool, tlsConfig *tls.Config) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		if !passHostHeader {
			req.Host = target.Host
		}
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		req.Header.Set("X-Real-IP", req.RemoteAddr)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", "https")
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}
	tr := &http.Transport{TLSClientConfig: tlsConfig}
	return &httputil.ReverseProxy{
		Director:  director,
		Transport: tr,
	}
}
