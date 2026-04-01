package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func ExampleNew() {
	handler, err := New(
		Join(
			PathByNameOfFixedTyped(strings.ToLower),
			JSONResponse(),
			Get(func(ctx context.Context, in struct {
				Users Fixed
			}) (string, error) {
				return "hello", nil
			}),
		),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/users", nil))

	fmt.Println(w.Code)
	fmt.Println(strings.TrimSpace(w.Body.String()))
	// Output:
	// 200
	// "hello"
}
