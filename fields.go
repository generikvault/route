package route

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"

	"slices"
)

// FieldOption configures how an input field is bound for a route.
type FieldOption[T any] func(route *route, name string, field reflect.Type) (fieldModifier[T], error)

type fieldModifier[T any] func(*request, T) (func(error) error, error)

// Fixed marks an input field that contributes a fixed path segment.
//
// Use it together with PathByNameOfFixedTyped.
type Fixed struct{}

// PathByNameOfFixedTyped binds Fixed fields as fixed path segments.
//
// The convert function maps field names to path segments.
func PathByNameOfFixedTyped(convert func(string) string) Option {
	return ByType(PathByName[*Fixed](convert))
}

// PathByName binds a field as a fixed path segment based on its field name.
//
// The convert function maps the field name to a path segment.
func PathByName[T any](convert func(string) string) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addFixedToPath(convert(name))
		return func(r *request, t T) (func(error) error, error) {
			r.popPath()
			return nil, nil
		}, nil
	}
}

// Path binds a field as the given fixed path segment.
func Path[T any](s string) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addFixedToPath(s)
		return func(r *request, t T) (func(error) error, error) {
			r.popPath()
			return nil, nil
		}, nil
	}
}

// StringPathIDs binds string fields from variable path segments.
//
// Use it with ByType(StringPathIDs()).
func StringPathIDs() FieldOption[*string] {
	return PathID(func(id string, v *string) error {
		*v = id
		return nil
	})
}

// IntPathIDs binds int fields from variable path segments.
//
// Use it with ByType(IntPathIDs()).
func IntPathIDs() FieldOption[*int] {
	return PathID(func(id string, v *int) error {
		i, err := strconv.Atoi(id)
		if err != nil {
			return err
		}
		*v = i
		return nil
	})
}

// PathID binds a field from one variable path segment.
func PathID[T any](f func(id string, v T) error) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addVarToPath()
		return func(r *request, v T) (func(error) error, error) {
			return nil, f(r.popPath(), v)
		}, nil
	}
}

// RequestValue binds a field from the current HTTP request.
func RequestValue[T any](f func(r *http.Request, v T) error) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		return func(r *request, v T) (func(error) error, error) {
			return nil, f(r.Request, v)
		}, nil
	}
}

// ClosableRequestValue binds a field from the request and returns an optional closer.
//
// The closer runs after request handling and receives the current error state.
func ClosableRequestValue[T any](f func(r *http.Request, v T) (func(error) error, error)) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		return func(r *request, v T) (func(error) error, error) {
			return f(r.Request, v)
		}, nil
	}
}

// ByName binds a specific input field name using the given field options.
func ByName(name string, opts ...FieldOption[any]) Option {
	return func(r *router) error {
		r.addNameRouteOption(name, func(route *route, name string, field reflect.Type) (fieldModifier[any], error) {
			return combinedFieldModifier(opts, route, name, field)
		})
		return nil
	}
}

// ByType binds fields of type T using the given field options.
func ByType[T any](opts ...FieldOption[*T]) Option {
	return func(r *router) error {
		r.addTypeRouteOption(typeOf[T](), func(route *route, name string, field reflect.Type) (fieldModifier[any], error) {
			return combinedFieldModifier(opts, route, name, field)
		})
		return nil
	}
}

func combinedFieldModifier[T any](opts []FieldOption[T], route *route, name string, field reflect.Type) (fieldModifier[any], error) {
	mods := make([]fieldModifier[T], 0, len(opts))
	for _, opt := range opts {
		mod, err := opt(route, name, field)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			mods = append(mods, mod)
		}
	}

	mods = slices.Clip(mods)
	return func(r *request, v any) (close func(error) error, err error) {
		closers := make([]func(error) error, 0, len(mods))
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recover panic: %v", r)
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
			}

			for _, closer := range slices.Backward(closers) {
				if inner := closer(err); inner != nil && err == nil {
					err = inner
				}
			}
		}()
		for _, mod := range mods {
			closer, err := mod(r, v.(T))
			if err != nil {
				return nil, err
			}
			if closer != nil {
				closers = append(closers, closer)
			}
		}
		if len(closers) == 0 {
			return nil, nil
		}
		delayed := closers
		closers = nil
		if len(delayed) == 1 {
			return delayed[0], nil
		}
		return func(err error) error {
			var inner error
			for _, closer := range slices.Backward(delayed) {
				if err := closer(err); err != nil && inner == nil {
					inner = err
				}
			}
			return inner
		}, nil
	}, nil
}
