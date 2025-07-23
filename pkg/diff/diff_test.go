package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareJSON_Wildcards(t *testing.T) {
	t.Parallel()

	// Test cases for string wildcards
	require.NoError(t, CompareJSON(`{"key": "%s"}`, `{"key": "any string"}`))
	require.Error(t, CompareJSON(`{"key": "%s"}`, `{"key": 123}`))
	require.NoError(t, CompareJSON(`{"key": "%S"}`, `{"key": "any string"}`))
	require.NoError(t, CompareJSON(`{"key": "%S"}`, `{"key": null}`))
	require.Error(t, CompareJSON(`{"key": "%S"}`, `{"key": 123}`))

	// Test cases for time wildcards
	require.NoError(t, CompareJSON(`{"key": "%datetime"}`, `{"key": "2024-01-01T12:34:56Z"}`))
	require.Error(t, CompareJSON(`{"key": "%datetime"}`, `{"key": "not a date"}`))
	require.NoError(t, CompareJSON(`{"key": "%date"}`, `{"key": "2024-01-01"}`))
	require.Error(t, CompareJSON(`{"key": "%date"}`, `{"key": "not a date"}`))

	// Test cases for combined wildcards
	require.NoError(t, CompareJSON(`{"key": "value-%s-suffix"}`, `{"key": "value-any-suffix"}`))
}

func TestToRegexp(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expectedRegexp string
		format         string
	}{
		{`%`, `%%`},
		{`foo bar`, `foo bar`},
		{`\d+`, `%d`},
		{`[+-]?\d+`, `%i`},
		{`\d+\.\d+`, `%d.%d`},
		{`[0-9a-fA-F]+`, `%x`},
		{`[-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?`, `%f`},
		{`.+`, `%s`},
		{`.*`, `%S`},
		{`(.|\n)+`, `%a`},
		{`(.|\n)*`, `%A`},
		{`\s*`, `%w`},
		{`.`, `%c`},
		{`foo/bar`, `foo%ebar`},
	}
	for _, c := range cases {
		require.Equal(t, c.expectedRegexp, toRegexp(c.format), "format: "+c.format)
	}
}

func TestCompareJSON_ReorderedColumns(t *testing.T) {
	t.Parallel()

	// Same data, different column order
	jsonA := `{"a":1,"b":2,"c":3}`
	jsonB := `{"c":3,"b":2,"a":1}`
	require.NoError(t, CompareJSON(jsonA, jsonB))

	// Data mismatch, different column order
	jsonC := `{"a":1,"b":2,"c":3}`
	jsonD := `{"c":3,"b":999,"a":1}`
	require.Error(t, CompareJSON(jsonC, jsonD))
}
