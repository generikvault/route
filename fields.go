package route

import (
	"net/http"
	"reflect"
	"strconv"

	"slices"
)

// FieldOption configures the behavior to input field.
type FieldOption[T any] func(route *route, name string, field reflect.Type) (fieldModifier[T], error)

type fieldModifier[T any] func(*request, T) (func(), error)

// Fixed is a field type that can be used to trigger the PathByNameOfFixedTyped Option
// to add a fixed path segment to the route.
// The Option must be specified explicitly.
type Fixed struct{}

// PathByNameOfFixed returns an Option that adds a fixed path to the route.
// The Option is triggered by a Fixed field in the input struct.
func PathByNameOfFixedTyped(convert func(string) string) Option {
	return ByType(PathByName[*Fixed](convert))
}

// PathByName returns an FieldOption that adds a path segment based on the fields name to the route.
// The convert function is used to convert the field name to the path segment.
// For example to convert the path name to kebab case or append an s or just strings.ToLower.
func PathByName[T any](convert func(string) string) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addFixedToPath(convert(name))
		return func(r *request, t T) (func(), error) {
			r.popPath()
			return nil, nil
		}, nil
	}
}

// Path returns an FieldOption that adds given path segment to the route.
func Path[T any](s string) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addFixedToPath(s)
		return func(r *request, t T) (func(), error) {
			r.popPath()
			return nil, nil
		}, nil
	}
}

// StringPathIDs returns an FieldOption that enables the route to route string IDs.
// Call it with ByType(StringPathIDs()). Feel free to add surrounding FieldOptions.
func StringPathIDs() FieldOption[*string] {
	return PathID(func(id string, v *string) error {
		*v = id
		return nil
	})
}

// IntPathIDs returns an FieldOption that enables the route to route int IDs.
// Call it with ByType(IntPathIDs()). Feel free to add surrounding FieldOptions.
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

// PathID returns an FieldOption that adds an id to the path.
func PathID[T any](f func(id string, v T) error) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		route.addVarToPath()
		return func(r *request, v T) (func(), error) {
			return nil, f(r.popPath(), v)
		}, nil
	}
}

// RequestValue returns a FieldOption to modify the field based on the request.
func RequestValue[T any](f func(r *http.Request, v T) error) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		return func(r *request, v T) (func(), error) {
			return nil, f(r.Request, v)
		}, nil
	}
}

// ClosableRequestValue returns a FieldOption to modify the field based on the request.
// The returned function is called after the request is handled.
func ClosableRequestValue[T any](f func(r *http.Request, v T) (func(), error)) FieldOption[T] {
	return func(route *route, name string, field reflect.Type) (fieldModifier[T], error) {
		return func(r *request, v T) (func(), error) {
			return f(r.Request, v)
		}, nil
	}
}

// ByName returns an Option that sets the named field.
// For example form the request body or header or from URL getter variables.
func ByName(name string, opts ...FieldOption[any]) Option {
	return func(r *router) error {
		r.addNameRouteOption(name, func(route *route, name string, field reflect.Type) (fieldModifier[any], error) {
			mods := make([]fieldModifier[any], len(opts))
			for i, opt := range opts {
				mod, err := opt(route, name, field)
				if err != nil {
					return nil, err
				}
				mods[i] = mod
			}
			return combinedFieldModifier(mods), nil
		})
		return nil
	}
}

func combinedFieldModifier[T any](mods []fieldModifier[T]) fieldModifier[any] {
	mods = slices.Clip(mods)
	return func(r *request, v any) (func(), error) {
		closers := make([]func(), 0, len(mods))
		defer func() {
			for _, closer := range slices.Backward(closers) {
				closer()
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
		delayed := closers
		closers = nil
		return func() {
			for _, closer := range slices.Backward(delayed) {
				closer()
			}
		}, nil
	}
}

// ByType returns an Option that sets the typed field.
// For example form the request body, header, path or from URL getter variables.
// Via the given FieldOptions
func ByType[T any](opts ...FieldOption[*T]) Option {
	return func(r *router) error {
		r.addTypeRouteOption(typeOf[T](), func(route *route, name string, field reflect.Type) (fieldModifier[any], error) {
			mods := make([]fieldModifier[*T], 0, len(opts))
			for _, opt := range opts {
				mod, err := opt(route, name, field)
				if err != nil {
					return nil, err
				}
				if mod != nil {
					mods = append(mods, mod)
				}
			}
			return combinedFieldModifier(mods), nil
		})
		return nil
	}
}
