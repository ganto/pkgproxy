package pkgproxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

type bufferWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *bufferWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bufferWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bufferWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bufferWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
