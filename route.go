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
		if err := handleRoute(r, w, route, handler, router.responseEncoder); err != nil {
			router.HandleErr(r.Context(), w, fmt.Errorf("handling request: %w", err))
			return
		}
	})
	for _, middleware := range router.middleware {
		httpHandler = middleware(httpHandler)
	}
	route.node.handler = httpHandler
	return nil
}

func handleRoute[Input, Output any](r *http.Request, w http.ResponseWriter, route route, handler func(context.Context, Input) (Output, error), responseEncoder func(context.Context, http.ResponseWriter, any) error) (mErr error) {
	ctx := r.Context()
	var input Input

	defer func() {
		if r := recover(); r != nil && mErr == nil {
			mErr = fmt.Errorf("panic: %v", r)
		}
	}()

	inputValue := reflect.ValueOf(&input).Elem()

	path, err := splitPath(r.URL)
	if err != nil {
		return err
	}
	request := request{
		Request:  r,
		pathTail: path,
	}
	for i, fieldMod := range route.fields {
		field := inputValue.Field(i)
		close, err := fieldMod(&request, field.Addr().Interface())
		if err != nil {
			return fmt.Errorf("applying input option: %w", err)
		}
		if close != nil {
			defer func() {
				if r := recover(); r != nil && mErr == nil {
					mErr = fmt.Errorf("panic: %v", r)
				}
				if err := close(mErr); err != nil && mErr == nil {
					mErr = err
				}
			}()
		}
	}

	if r.Method == http.MethodHead {
		return
	}

	res, err := handler(ctx, input)
	if err != nil {
		return fmt.Errorf("handling request: %w", err)
	}

	if err := responseEncoder(ctx, w, res); err != nil {
		return fmt.Errorf("encoding response: %w", err)
	}

	return nil
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
