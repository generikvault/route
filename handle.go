package route

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

func New(opts ...Option) (http.HandlerFunc, error) {
	router := router{}
	for _, opt := range opts {
		if err := opt(&router); err != nil {
			return nil, err
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		path, err := splitPath(r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		handler, ok := router.Node(r.Method).Handler(path)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		handler.ServeHTTP(w, r)
	}, nil
}

func routeHandler[Input, Output any](router *router, node *node, handler func(context.Context, Input) (Output, error)) error {
	input := typeOf[Input]()

	route := route{
		node:   node,
		fields: make([]fieldModifier[any], input.NumField()),
	}

	for i := 0; i < input.NumField(); i++ {
		field := input.Field(i)
		if !field.IsExported() {
			return fmt.Errorf("field %s is not exported", field.Name)
		}
		if option, ok := router.routeOption(field); ok {
			option, err := option(&route, field.Name, field.Type)
			if err != nil {
				return err
			}
			route.fields[i] = option
			continue
		}

		return fmt.Errorf("no option for field %s type %s", field.Name, field.Type)
	}

	var httpHandler http.Handler
	httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		input, err := createInput[Input](r, route)
		if err != nil {
			router.HandleErr(ctx, w, fmt.Errorf("handling request: %w", err))
			return
		}

		if r.Method == http.MethodHead {
			return
		}

		res, err := handler(ctx, input)
		if err != nil {
			router.HandleErr(ctx, w, fmt.Errorf("handling request: %w", err))
			return
		}

		if err := router.responseEncoder(ctx, w, res); err != nil {
			router.HandleErr(ctx, w, fmt.Errorf("encoding response: %w", err))
			return
		}
	})
	for _, middleware := range router.middleware {
		httpHandler = middleware(httpHandler)
	}
	route.node.handler = httpHandler
	return nil
}

func createInput[Input any](r *http.Request, route route) (Input, error) {
	var input Input

	inputValue := reflect.ValueOf(&input).Elem()

	path, err := splitPath(r.URL)
	if err != nil {
		return input, err
	}
	request := request{
		Request:  r,
		pathTail: path,
	}
	for i, fieldMod := range route.fields {
		field := inputValue.Field(i)
		close, err := fieldMod(&request, field.Addr().Interface())
		if err != nil {
			return input, fmt.Errorf("applying input option: %w", err)
		}
		if close != nil {
			defer close()
		}
	}

	return input, nil
}

func splitPath(link *url.URL) ([]string, error) {
	if link.RawPath == "" {
		return strings.Split(link.Path, "/")[1:], nil
	}
	path := strings.Split(link.RawPath, "/")[1:]
	for i, p := range path {
		s, err := url.PathUnescape(p)
		if err != nil {
			return nil, fmt.Errorf("url.PathUnescape: %w", err)
		}
		path[i] = s
	}
	return path, nil
}

func Post[Input, Output any](handler func(context.Context, Input) (Output, error)) Option {
	return func(r *router) error {
		return routeHandler(r, &r.post, handler)
	}
}

func Put[Input, Output any](handler func(context.Context, Input) (Output, error)) Option {
	return func(r *router) error {
		return routeHandler(r, &r.put, handler)
	}
}

func Get[Input, Output any](handler func(context.Context, Input) (Output, error)) Option {
	return func(r *router) error {
		return routeHandler(r, &r.get, handler)
	}
}

func Delete[Input, Output any](handler func(context.Context, Input) (Output, error)) Option {
	return func(r *router) error {
		return routeHandler(r, &r.delete, handler)
	}
}
