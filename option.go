package connecthttp

type handlerConfig struct {
	drc DecodeRequestFunc
	enc EncodeResponseFunc
	ene EncodeErrorFunc
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
		c.drc = fn
	}
}

func WithEncodeResponseFunc(fn EncodeResponseFunc) HandlerOption {
	return func(c *handlerConfig) {
		c.enc = fn
	}
}

func WithEncodeErrorFunc(fn EncodeErrorFunc) HandlerOption {
	return func(c *handlerConfig) {
		c.ene = fn
	}
}
