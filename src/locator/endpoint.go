package locator

import (
	"fmt"
	"log"
	"net/url"
)

func NewEndpoint(backendURL *url.URL, passHostHeader bool) *StaticEndpoint {
	return &StaticEndpoint{
		backendURL:     backendURL,
		passHostHeader: passHostHeader,
	}
}

// StaticEndpoint holds information about a back-end HTTP(s) server.
type StaticEndpoint struct {
	// backendURL holds the URL of the back-end server.
	backendURL     *url.URL
	passHostHeader bool
}

func (e StaticEndpoint) URL() *url.URL {
	return e.backendURL
}

func (e StaticEndpoint) Address() string {
	port := e.backendURL.Port()
	if port == "" {
		if e.backendURL.Scheme == "https" || e.backendURL.Scheme == "wss" {
			port = "443"
		} else {
			port = "80"
		}
	}
	address := fmt.Sprintf("%s:%s", e.backendURL.Hostname(), port)
	log.Printf("Address: %s", address)
	return address
}

func (e StaticEndpoint) IsTLS() bool {
	return (e.backendURL.Scheme == "https" || e.backendURL.Scheme == "wss")
}

func (e StaticEndpoint) PassHostHeader() bool {
	return e.passHostHeader
}
