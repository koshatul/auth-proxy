package proxy

import (
	"net/url"
)

// Endpoint is an interface that is used to represent a backend service.
type Endpoint interface {
	Address() string
	URL() *url.URL
	IsTLS() bool
	PassHostHeader() bool
}
