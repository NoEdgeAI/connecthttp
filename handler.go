package connecthttp

import (
	"context"
	"fmt"
	"net/http"
)

type Handler struct {
	config         *handlerConfig
	implementation func(context.Context, HandlerConn) error
}

func NewHandler[Req, Res any](
	procedure string,
	unary func(context.Context, *Request[Req]) (*Response[Res], error),
	options ...HandlerOption,
) *Handler {
	untyped := func(ctx context.Context, request AnyRequest) (AnyResponse, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		typed, ok := request.(*Request[Req])
		if !ok {
			return nil, fmt.Errorf("unexpected handler request type %T", request)
		}
		return unary(ctx, typed)
	}

	implementation := func(ctx context.Context, conn HandlerConn) error {
		var msg Req
		if err := conn.Receive(&msg); err != nil {
			return err
		}

		response, err := untyped(ctx, &Request[Req]{Msg: &msg})
		if err != nil {
			return err
		}

		return conn.Send(response.Any())
	}

	return &Handler{
		config:         newHandlerConfig(options...),
		implementation: implementation,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn := &handlerConn{r: r, w: w, config: h.config}
	if err := h.implementation(r.Context(), conn); err != nil {
		h.config.ene(w, r, err)
		return
	}
}
