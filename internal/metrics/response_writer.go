package metrics

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// websocketResponseWriter
type websocketResponseWriter struct {
	http.ResponseWriter
}

func (w *websocketResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}
