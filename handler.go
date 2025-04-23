package connecthttp

import (
	"context"
	"net/http"
)

type Handler struct {
	impl func(w http.ResponseWriter, r *http.Request)
}

func NewHandler[Req, Res any](
	_ string,
	unary func(context.Context, *Request[Req]) (*Response[Res], error),
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(options...)
	impl := func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := config.drf(r, &req); err != nil {
			config.eef(w, r, err)
			return
		}

		response, err := unary(r.Context(), NewRequest(&req))
		if err != nil {
			config.eef(w, r, err)
			return
		}

		if err := config.erf(w, r, response.Any()); err != nil {
			config.eef(w, r, err)
			return
		}
	}

	return &Handler{
		impl: impl,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.impl(w, r)
}
