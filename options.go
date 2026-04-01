package route

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Option configures router behavior.
type Option func(*router) error

// Join combines multiple options into one option.
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

// ResponseEncoder sets the encoder used for handler outputs.
//
// The encoder can switch behavior based on the concrete output value.
func ResponseEncoder(encoder func(context.Context, http.ResponseWriter, *http.Request, any) error) Option {
	return func(r *router) error {
		r.responseEncoder = encoder
		return nil
	}
}

// Body binds a field from the request body using the provided decoder.
func Body(decoder func(io.Reader, any) error) FieldOption[any] {
	return RequestValue(func(r *http.Request, value any) error {
		return decoder(r.Body, value)
	})
}

// JSONBody binds a field from a JSON request body.
func JSONBody() FieldOption[any] {
	return Body(func(r io.Reader, i any) error {
		return json.NewDecoder(r).Decode(i)
	})
}

// JSONResponse encodes handler output as JSON.
func JSONResponse() Option {
	return ResponseEncoder(func(ctx context.Context, w http.ResponseWriter, r *http.Request, v any) error {
		return json.NewEncoder(w).Encode(v)
	})
}

// HandleError sets the error handler for route and binding failures.
func HandleError(handleErr func(ctx context.Context, w http.ResponseWriter, err error)) Option {
	return func(r *router) error {
		r.handleErr = handleErr
		return nil
	}
}

// Middleware appends middleware to all registered handlers.
func Middleware(middleware ...func(http.Handler) http.Handler) Option {
	return func(r *router) error {
		r.middleware = append(r.middleware, middleware...)
		return nil
	}
}
