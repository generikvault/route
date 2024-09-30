package route

import (
	"context"
	"net/http"
	"reflect"
)

type router struct {
	get    node
	post   node
	put    node
	delete node

	nameRouteOptions map[string]FieldOption[any]
	typeRouteOptions map[reflect.Type]FieldOption[any]

	responseEncoder func(context.Context, http.ResponseWriter, any) error

	handleErr func(context.Context, http.ResponseWriter, error)

	middleware []func(http.Handler) http.Handler
}

func (r *router) Node(method string) node {
	switch method {
	case http.MethodHead:
		fallthrough
	case http.MethodGet:
		return r.get
	case http.MethodPost:
		return r.post
	case http.MethodPut:
		return r.put
	case http.MethodDelete:
		return r.delete
	default:
		return node{}
	}
}

func (r *router) HandleErr(ctx context.Context, w http.ResponseWriter, err error) {
	if r.handleErr != nil {
		r.handleErr(ctx, w, err)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (r *router) addTypeRouteOption(t reflect.Type, option FieldOption[any]) {
	if r.typeRouteOptions == nil {
		r.typeRouteOptions = make(map[reflect.Type]FieldOption[any])
	}
	r.typeRouteOptions[t] = option
}

func (r *router) addNameRouteOption(name string, option FieldOption[any]) {
	if r.nameRouteOptions == nil {
		r.nameRouteOptions = make(map[string]FieldOption[any])
	}
	r.nameRouteOptions[name] = option
}

func (r *router) routeOption(field reflect.StructField) (FieldOption[any], bool) {
	if named, ok := r.nameRouteOptions[field.Name]; ok {
		return named, true
	}

	if typed, ok := r.typeRouteOptions[field.Type]; ok {
		return typed, true
	}
	return nil, false
}

type route struct {
	*node
	fields []fieldModifier[any]
}

func (r *route) addFixedToPath(name string) {
	next, ok := r.childs[name]
	if !ok {
		if r.childs == nil {
			r.childs = make(map[string]*node)
		}
		next = &node{}
		r.childs[name] = next
	}
	r.node = next
}

func (r *route) addVarToPath() {
	next := r.child
	if next == nil {
		next = &node{}
		r.child = next
	}
	r.node = next
}

type request struct {
	*http.Request
	pathTail []string
}

func (r *request) popPath() string {
	s := r.pathTail[0]
	r.pathTail = r.pathTail[1:]
	return s
}

func typeOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(t)
}
