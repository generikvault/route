package route

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		opt         Option
		req         *http.Request
		body        string
		requestCode int
		wantErr     bool
	}{
		{
			name: "GET",
			opt: Join(
				PathByNameOfFixedTyped(strings.ToLower),
				JSONResponse(),
				Get(func(ctx context.Context, in struct {
					Foo Fixed
				}) (string, error) {
					return "Hello World", nil
				}),
			),
			req:         httptest.NewRequest("GET", "http://example.com/foo", nil),
			body:        `"Hello World"`,
			requestCode: http.StatusOK,
		},
		{
			name: "404",
			opt: Join(
				PathByNameOfFixedTyped(strings.ToLower),
				JSONResponse(),
				Get(func(ctx context.Context, in struct {
					Foo Fixed
				}) (string, error) {
					return "Hello World", nil
				}),
			),
			req:         httptest.NewRequest("GET", "http://example.com/fooo", nil),
			requestCode: http.StatusNotFound,
		},
		{
			name: "POST",
			opt: testOptions(
				Post(func(ctx context.Context, in struct {
					Body struct{ Greetings string }
				}) (string, error) {
					return in.Body.Greetings, nil
				}),
			),
			req:         httptest.NewRequest("POST", "http://example.com", strings.NewReader(`{"Greetings":"Hello Body"}`)),
			body:        `"Hello Body"`,
			requestCode: http.StatusOK,
		},
		{
			name: "IDs",
			opt: testOptions(
				Get(func(ctx context.Context, in struct {
					IntID    int
					Stuff    Fixed
					StringID string
				}) (string, error) {
					return fmt.Sprintf("%d times Hello %s", in.IntID, in.StringID), nil
				}),
			),
			req: httptest.NewRequest("GET",
				"http://example.com/7/stuff/%2FWorld",
				nil),
			body:        `"7 times Hello /World"`,
			requestCode: http.StatusOK,
		},
		{
			name: "private-fields",
			opt: testOptions(
				Get(func(ctx context.Context, in struct {
					private int
				}) (string, error) {
					return "Hello World", nil
				}),
			),
			req:     httptest.NewRequest("GET", "http://example.com/7/stuff/World", nil),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := New(tt.opt)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			if err != nil {
				t.Errorf("New() error = %v", err)
				return
			}

			w := httptest.NewRecorder()
			handler(w, tt.req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.requestCode, resp.StatusCode)
			if tt.body != "" {
				assert.Equal(t, tt.body, strings.TrimSpace(string(body)))
			}
		})
	}
}

func testOptions(opts ...Option) Option {
	return Join(
		append(
			[]Option{
				JSONResponse(),
				ByName("Body", JSONBody()),
				PathByNameOfFixedTyped(strings.ToLower),
				ByType(IntPathIDs()),
				ByType(StringPathIDs()),
			},
			opts...,
		)...,
	)
}

func TestClosableRequestValue(t *testing.T) {
	value := "Hello World"

	handler, err := New(
		JSONResponse(),
		ByType(ClosableRequestValue(func(r *http.Request, v **string) (func(error), error) {
			*v = &value
			return func(error) {
				value = "Goodbye World"
			}, nil
		})),
		Get(func(ctx context.Context, in struct {
			V *string
		}) (string, error) {
			return *in.V, nil
		}),
	)

	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}

	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "http://example.com", nil))

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, 200, resp.StatusCode)
	unquoted, err := strconv.Unquote(strings.TrimSpace(string(body)))
	if err != nil {
		t.Errorf("strconv.Unquote() error = %v", err)
		return
	}
	assert.Equal(t, "Hello World", unquoted)
	assert.Equal(t, "Goodbye World", value)
}

func TestIterDefer(t *testing.T) {
	var values []int
	func() {
		for _, i := range slices.Backward([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
			defer func() { values = append(values, i) }()
		}
	}()
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, values)

}
