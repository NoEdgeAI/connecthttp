package connecthttp

type Request[T any] struct {
	Msg *T
}

func (r *Request[_]) Any() any {
	return r.Msg
}

type AnyRequest interface {
	Any() any
}

func NewRequest[T any](message *T) *Request[T] {
	return &Request[T]{
		Msg: message,
	}
}

type Response[T any] struct {
	Msg *T
}

func (r *Response[_]) Any() any {
	return r.Msg
}

type AnyResponse interface {
	Any() any
}

func NewResponse[T any](message *T) *Response[T] {
	return &Response[T]{
		Msg: message,
	}
}
