package route

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Option is a function that sets a router option.
type Option func(*router) error

// Join returns an Option that joins multiple options.
func Join(opts ...Option) Option {
	return func(r *router) error {
		for _, opt := range opts {
			if err := opt(r); err != nil {
				return err
			}
		}
		return nil
	}
}

// ResponseEncoder returns an Option that sets the response encoder.
// Different Output types can be handled differently by the given encoder Function.
func ResponseEncoder(encoder func(context.Context, http.ResponseWriter, any) error) Option {
	return func(r *router) error {
		r.responseEncoder = encoder
		return nil
	}
}

// Body returns an FieldOption that decodes the request body into the field.
func Body(decoder func(io.Reader, any) error) FieldOption[any] {
	return RequestValue[any](func(r *http.Request, value any) error {
		return decoder(r.Body, value)
	})
}

// JSONBody returns an FieldOption that decodes the request body as JSON into the field.
func JSONBody() FieldOption[any] {
	return Body(func(r io.Reader, i any) error {
		return json.NewDecoder(r).Decode(i)
	})
}

// JSONResponse returns an Option that encodes the response as JSON.
func JSONResponse() Option {
	return ResponseEncoder(func(ctx context.Context, w http.ResponseWriter, v any) error {
		return json.NewEncoder(w).Encode(v)
	})
}

// HandleError returns an Option that sets the error handler.
func HandleError(handleErr func(ctx context.Context, w http.ResponseWriter, err error)) Option {
	return func(r *router) error {
		r.handleErr = handleErr
		return nil
	}
}

// Middleware returns an Option that adds given middleware.
func Middleware(middleware ...func(http.Handler) http.Handler) Option {
	return func(r *router) error {
		r.middleware = append(r.middleware, middleware...)
		return nil
	}
}
