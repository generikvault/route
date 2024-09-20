package route

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
			req:         httptest.NewRequest("GET", "http://example.com/7/stuff/World", nil),
			body:        `"7 times Hello World"`,
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
