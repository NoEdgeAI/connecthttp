package connecthttp

import "net/http"

type Codec interface {
	func(*http.Request, any) error
	func(http.ResponseWriter, *http.Request, any) error
	func(http.ResponseWriter, *http.Request, error)
}

// DecodeRequestFunc is decode request func.
type DecodeRequestFunc func(*http.Request, any) error

// EncodeResponseFunc is encode response func.
type EncodeResponseFunc func(http.ResponseWriter, *http.Request, any) error

// EncodeErrorFunc is encode error func.
type EncodeErrorFunc func(http.ResponseWriter, *http.Request, error)
