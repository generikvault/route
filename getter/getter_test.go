package getter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntoStruct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?a=1&b=2&c=hello&d=a&d=b&f=abc", nil)
	type testStruct struct {
		A   int
		B   *int
		C   string
		D   []string
		Fom string `getter:"f"`
		X   *int
	}
	var s testStruct
	require.NoError(t, IntoStruct(r, &s))
	assert.Equalf(t, 1, s.A, "expected 1, got %d", s.A)
	require.NotNilf(t, s.B, "expected not nil, got nil")
	assert.EqualValues(t, 2, *s.B, "expected 2, got %v", s.B)
	assert.Equalf(t, "hello", s.C, "expected hello, got %s", s.C)
	assert.Equalf(t, []string{"a", "b"}, s.D, "expected [a b], got %v", s.D)
	assert.Equalf(t, "abc", s.Fom, "expected abc, got %s", s.Fom)
	assert.Nilf(t, s.X, "expected nil, got %v", s.X)
}
