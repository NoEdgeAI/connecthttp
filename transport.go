package connecthttp

import (
	"context"
	"net/http"
)

type (
	transportKey struct{}
)

type Transport struct {
	request  *http.Request
	response http.ResponseWriter
}

func (tr *Transport) Request() *http.Request {
	return tr.request
}

func (tr *Transport) Response() http.ResponseWriter {
	return tr.response
}

func TransportFromContext(ctx context.Context) (tr *Transport, ok bool) {
	tr, ok = ctx.Value(transportKey{}).(*Transport)
	return
}

func NewTransportContext(ctx context.Context, tr *Transport) context.Context {
	return context.WithValue(ctx, transportKey{}, tr)
}
