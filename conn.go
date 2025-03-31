package connecthttp

import "net/http"

type HandlerConn interface {
	Receive(any) error
	Send(any) error
}

type handlerConn struct {
	r      *http.Request
	w      http.ResponseWriter
	config *handlerConfig
}

func (h *handlerConn) Receive(a any) error {
	return h.config.drc(h.r, a)
}

func (h *handlerConn) Send(a any) error {
	return h.config.enc(h.w, h.r, a)
}
