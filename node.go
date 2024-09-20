package route

import "net/http"

type node struct {
	childs         map[string]*node
	child          *node
	allowRemainder bool
	handler        http.Handler
}

func (n node) Handler(path []string) (http.Handler, bool) {
	if len(path) == 0 {
		return n.handler, n.handler != nil
	}
	first := path[0]
	if child, ok := n.childs[first]; ok {
		if handler, ok := child.Handler(path[1:]); ok {
			return handler, true
		}
	}
	if n.child != nil {
		return n.child.Handler(path[1:])
	}
	if n.allowRemainder {
		return n.handler, true
	}
	return nil, false
}
