package connecthttp

type handlerConfig struct {
	drf DecodeRequestFunc
	erf EncodeResponseFunc
	eef EncodeErrorFunc
}

type HandlerOption func(*handlerConfig)

func newHandlerConfig(options ...HandlerOption) *handlerConfig {
	config := &handlerConfig{}
	for _, opt := range options {
		opt(config)
	}

	return config
}

func WithDecodeRequestFunc(fn DecodeRequestFunc) HandlerOption {
	return func(c *handlerConfig) {
		c.drf = fn
	}
}

func WithEncodeResponseFunc(fn EncodeResponseFunc) HandlerOption {
	return func(c *handlerConfig) {
		c.erf = fn
	}
}

func WithEncodeErrorFunc(fn EncodeErrorFunc) HandlerOption {
	return func(c *handlerConfig) {
		c.eef = fn
	}
}
