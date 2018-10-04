package proxy

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/koshatul/auth-proxy/src/statuspage"
)

// Handler is an http.Handler that proxies requests to an upstream server.
type Handler struct {
	HTTPProxy        Proxy
	WebSocketProxy   Proxy
	Backend          Endpoint
	StatusPageWriter statuspage.Writer
	Logger           *log.Logger
}

// ServeHTTP proxies the request to the appropriate upstream server.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logContext := &LogContext{Logger: handler.Logger, Request: request}
	logContext.Metrics.Start()

	err := handler.forward(writer, request, logContext)

	// If there was an error and no response has been sent, send an error page.
	if err != nil && logContext.StatusCode == 0 {
		handler.writeStatusPage(writer, request, logContext, err)
	}

	logContext.Log(err)
}

func (handler *Handler) forward(
	writer http.ResponseWriter,
	request *http.Request,
	logContext *LogContext,
) (err error) {
	isWebSocket := isWebSocketUpgrade(request.Header)
	logContext.IsWebSocket = isWebSocket

	var proxy Proxy
	if isWebSocket {
		proxy = handler.WebSocketProxy
	} else {
		proxy = handler.HTTPProxy
	}

	return proxy.Forward(
		writer,
		request,
		handler.prepareUpstreamRequest(request, handler.Backend, isWebSocket),
		handler.Backend,
		logContext,
	)
}

// prepareUpstreamRequest makes a new http.Request that uses the given endpoint
// as the upstream server.
func (handler *Handler) prepareUpstreamRequest(
	request *http.Request,
	backend Endpoint,
	isWebSocket bool,
) *http.Request {
	upstreamRequest := *request
	upstreamRequest.Header = handler.prepareUpstreamHeaders(request, isWebSocket)

	upstreamURL := *request.URL
	upstreamURL.Host = backend.Address()

	if isWebSocket {
		if backend.IsTLS() {
			upstreamURL.Scheme = "wss"
		} else {
			upstreamURL.Scheme = "ws"
		}
	} else {
		if backend.IsTLS() {
			upstreamURL.Scheme = "https"
		} else {
			upstreamURL.Scheme = "http"
		}
	}

	upstreamRequest.URL = &upstreamURL

	return &upstreamRequest
}

// prepareUpstreamHeaders produces a copy of request.Header and modifies them so
// that they are suitable to send to the upstream server.
func (handler *Handler) prepareUpstreamHeaders(request *http.Request, isWebSocket bool) http.Header {
	upstreamHeaders := http.Header{}
	forwardedFor, _, _ := net.SplitHostPort(request.RemoteAddr)

	for name, values := range request.Header {
		if name == "X-Forwarded-For" {
			forwardedFor = strings.Join(values, ", ") + ", " + forwardedFor
		} else if !isHopByHopHeader(name) {
			upstreamHeaders[name] = values
		}
	}

	upstreamHeaders.Set("Host", request.Host)
	upstreamHeaders.Set("X-Forwarded-For", forwardedFor)
	upstreamHeaders.Set("X-Forwarded-SSL", "on")

	if isWebSocket {
		upstreamHeaders.Set("X-Forwarded-Proto", "wss")
	} else {
		upstreamHeaders.Set("X-Forwarded-Proto", "https")
	}

	return upstreamHeaders
}

// writeStatusPage responds with a status page for the given error.
func (handler *Handler) writeStatusPage(
	writer http.ResponseWriter,
	request *http.Request,
	logContext *LogContext,
	err error,
) {
	statusWriter := handler.StatusPageWriter
	if statusWriter == nil {
		statusWriter = statuspage.DefaultWriter
	}

	logContext.Metrics.FirstByteSent()
	defer logContext.Metrics.LastByteSent()

	logContext.StatusCode, logContext.Metrics.BytesOut, _ = statusWriter.WriteError(
		writer,
		request,
		err,
	)
}
