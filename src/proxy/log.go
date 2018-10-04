package proxy

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/golang/gddo/httputil/header"
)

// LogContext holds information about an HTTP request/response transaction used
// for logging.
type LogContext struct {
	Logger      *log.Logger
	StatusCode  int
	IsWebSocket bool
	Metrics     Metrics
	Request     *http.Request

	prefixLength int
	buffer       bytes.Buffer
}

// Log writes a log entry for the context to the logger.
//
// The log format consists of the following space separated fields:
//
// - remote address
// - frontent address
// - backend address
// - backend description
// - request information (method, URI and protocol)
// - event type
// - http status code
// - time to first byte
// - time to last byte
// - bytes inbound
// - bytes outbound
// - message (optional)
//
// The event types are:
// - "HTTP" - regular HTTP request
// - "WS/CN" - websocket connected
// - "WS/DC" - websocket disconnected
//
// All fields are always present, except for the message which is optional. If a
// field value is unknown or not applicable, a hyphen is used in place. If a
// field value contains spaces or other special characters it is rendered as a
// double-quoted Go string. This allows log output to be parsed programatically.
func (ctx *LogContext) Log(err error) {
	if ctx.Logger == nil || ctx.isMuted() {
		return
	}

	ctx.writePrefix()

	// event type
	if !ctx.IsWebSocket {
		ctx.write("HTTP")
	} else if ctx.Metrics.IsLastByteSent() {
		ctx.write("WS/DC")
	} else {
		ctx.write("WS/CN")
	}

	// status code
	if ctx.StatusCode == 0 {
		ctx.write("")
	} else {
		ctx.write("%d", ctx.StatusCode)
	}

	// time to first byte
	if ctx.Metrics.IsFirstByteSent() {
		ctx.write(
			"f/%sms",
			humanize.FormatFloat("#,###.##", ctx.Metrics.TimeToFirstByte),
		)
	} else {
		ctx.write("")
	}

	// time to last byte
	if ctx.Metrics.IsLastByteSent() {
		ctx.write(
			"l/%sms",
			humanize.FormatFloat("#,###.##", ctx.Metrics.TimeToLastByte),
		)

		// bytes in
		ctx.write(
			"i/%s",
			humanize.FormatFloat("#,###.", float64(ctx.Metrics.BytesIn)),
		)

		// bytes out
		ctx.write(
			"o/%s",
			humanize.FormatFloat("#,###.", float64(ctx.Metrics.BytesOut)),
		)
	} else {
		ctx.write("")
		ctx.write("")
		ctx.write("")
	}

	// optional message
	if err != nil {
		ctx.write(err.Error())
	}

	ctx.Logger.Println(ctx.buffer.String())
	ctx.buffer.Truncate(ctx.prefixLength)
}

// write is a helper function that writes to a string to a buffer, quoting the
// string if it contains whitespace or special characters.
func (ctx *LogContext) write(str string, v ...interface{}) {
	if ctx.buffer.Len() != 0 {
		ctx.buffer.WriteRune(' ')
	}

	if len(v) != 0 {
		str = fmt.Sprintf(str, v...)
	}

	if str == "" {
		ctx.buffer.WriteRune('-')
		return
	}

	if strings.ContainsAny(str, " \a\b\f\n\r\t\v\"") {
		ctx.buffer.WriteString(strconv.Quote(str))
	} else {
		ctx.buffer.WriteString(str)
	}
}

func (ctx *LogContext) writePrefix() {
	if ctx.prefixLength != 0 {
		return
	}

	// remote address
	var remoteAddr string
	for _, ip := range header.ParseList(ctx.Request.Header, "X-Forwarded-For") {
		remoteAddr += ip + ","
	}
	ctx.write(remoteAddr + ctx.Request.RemoteAddr)

	// frontend
	ctx.write(ctx.Request.Host)

	// request information
	ctx.write(
		"%s %s %s",
		ctx.Request.Method,
		ctx.Request.URL.RequestURI(),
		ctx.Request.Proto,
	)

	ctx.prefixLength = ctx.buffer.Len()
}

func (ctx *LogContext) isMuted() bool {
	if ctx.Request.URL.Path != "/favicon.ico" {
		return false
	}

	return 200 <= ctx.StatusCode && ctx.StatusCode < 500
}
