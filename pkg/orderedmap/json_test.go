package orderedmap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderedMap_MarshalJSON(t *testing.T) {
	t.Parallel()
	o := New()

	// number
	o.Set("number", 3)
	// string
	o.Set("string", "x")
	// string
	o.Set("specialstring", "\\.<>[]{}_-")
	// new value keeps key in old position
	o.Set("number", 4)
	// keys not sorted alphabetically
	o.Set("z", 1)
	o.Set("a", 2)
	o.Set("b", 3)
	// slice
	o.Set("slice", []any{
		"1",
		1,
	})
	// orderedmap
	v := New()
	v.Set("e", 1)
	v.Set("a", 2)
	o.Set("orderedmap", v)
	// escape key
	o.Set("test\n\r\t\\\"ing", 9)

	// result
	out, err := json.Marshal(o)
	assert.NoError(t, err)
	assert.Equal(t, `{"number":4,"string":"x","specialstring":"\\.\u003c\u003e[]{}_-","z":1,"a":2,"b":3,"slice":["1",1],"orderedmap":{"e":1,"a":2},"test\n\r\t\\\"ing":9}`, string(out))

	// result with indent
	expected := `{
  "number": 4,
  "string": "x",
  "specialstring": "\\.\u003c\u003e[]{}_-",
  "z": 1,
  "a": 2,
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a": 2
  },
  "test\n\r\t\\\"ing": 9
}`
	out, err = json.MarshalIndent(o, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expected, string(out))
}

func TestOrderedMap_MarshalJSON_Blank(t *testing.T) {
	t.Parallel()
	o := New()

	// blank map
	out, err := json.Marshal(o)
	assert.NoError(t, err)
	assert.Equal(t, "{}", string(out))

	// blank map with indent
	out, err = json.MarshalIndent(o, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, "{}", string(out))
}

func TestOrderedMap_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	in := `{
  "number": 4,
  "string": "x",
  "z": 1,
  "a": "should not break with unclosed { character in value",
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a { nested key with brace": "with a }}}} }} {{{ brace value",
	"after": {
		"link": "test {{{ with even deeper nested braces }"
	}
  },
  "test\"ing": 9,
  "after": 1,
  "multitype_array": [
    "test",
	1,
	{ "map": "obj", "it" : 5, ":colon in key": "colon: in value" },
	[{"inner": "map"}]
  ],
  "should not break with { character in key": 1
}`
	o := New()
	err := json.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "number",
			Value: float64(4),
		},
		{
			Key:   "string",
			Value: "x",
		},
		{
			Key:   "z",
			Value: float64(1),
		},
		{
			Key:   "a",
			Value: "should not break with unclosed { character in value",
		},
		{
			Key:   "b",
			Value: float64(3),
		},
		{
			Key: "slice",
			Value: []any{
				"1",
				float64(1),
			},
		},
		{
			Key: "orderedmap",
			Value: FromPairs([]Pair{
				{
					Key:   "e",
					Value: float64(1),
				},
				{
					Key:   "a { nested key with brace",
					Value: "with a }}}} }} {{{ brace value",
				},
				{
					Key: "after",
					Value: FromPairs([]Pair{
						{
							Key:   "link",
							Value: "test {{{ with even deeper nested braces }",
						},
					}),
				},
			}),
		},
		{
			Key:   "test\"ing",
			Value: float64(9),
		},
		{
			Key:   "after",
			Value: float64(1),
		},
		{
			Key: "multitype_array",
			Value: []any{
				"test",
				float64(1),
				FromPairs([]Pair{
					{
						Key:   "map",
						Value: "obj",
					},
					{
						Key:   "it",
						Value: float64(5),
					},
					{
						Key:   ":colon in key",
						Value: "colon: in value",
					},
				}),
				[]any{
					FromPairs([]Pair{
						{
							Key:   "inner",
							Value: "map",
						},
					}),
				},
			},
		},
		{
			Key:   "should not break with { character in key",
			Value: float64(1),
		},
	}), o)
}

func TestOrderedMap_UnmarshalJSON_DuplicateKeys(t *testing.T) {
	t.Parallel()
	in := `{
		"a": [{}, []],
		"b": {"x":[1]},
		"c": "x",
		"d": {"x":1},
		"b": [{"x":[]}],
		"c": 1,
		"d": {"y": 2},
		"e": [{"x":1}],
		"e": [[]],
		"e": [{"z":2}],
		"a": {},
		"b": [[1]]
	}`

	o := New()
	err := json.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "c",
			Value: float64(1),
		},
		{
			Key: "d",
			Value: FromPairs([]Pair{
				{
					Key:   "y",
					Value: float64(2),
				},
			}),
		},
		{
			Key: "e",
			Value: []any{FromPairs([]Pair{
				{
					Key:   "z",
					Value: float64(2),
				},
			})},
		},
		{
			Key:   "a",
			Value: New(),
		},
		{
			Key:   "b",
			Value: []any{[]any{float64(1)}},
		},
	}), o)
}

func TestOrderedMap_UnmarshalJSON_SpecialChars(t *testing.T) {
	t.Parallel()
	in := `{ " \u0041\n\r\t\\\\\\\\ ": { "\\\\\\": "\\\\\"\\" }, "\\":  " \\\\ test ", "\n": "\r" }`

	o := New()
	err := json.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key: " \u0041\n\r\t\\\\\\\\ ",
			Value: FromPairs([]Pair{
				{
					Key:   "\\\\\\",
					Value: "\\\\\"\\",
				},
			}),
		},
		{
			Key:   "\\",
			Value: " \\\\ test ",
		},
		{
			Key:   "\n",
			Value: "\r",
		},
	}), o)
}

func TestOrderedMap_UnmarshalJSON_ArrayOfMaps(t *testing.T) {
	t.Parallel()
	in := `
{
  "name": "test",
  "percent": 6,
  "breakdown": [
    {
      "name": "a",
      "percent": 0.9
    },
    {
      "name": "b",
      "percent": 0.9
    },
    {
      "name": "d",
      "percent": 0.4
    },
    {
      "name": "e",
      "percent": 2.7
    }
  ]
}
`
	o := New()
	err := json.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "name",
			Value: "test",
		},
		{
			Key:   "percent",
			Value: float64(6),
		},
		{
			Key: "breakdown",
			Value: []any{
				FromPairs([]Pair{
					{Key: "name", Value: "a"},
					{Key: "percent", Value: float64(0.9)},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "b"},
					{Key: "percent", Value: float64(0.9)},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "d"},
					{Key: "percent", Value: float64(0.4)},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "e"},
					{Key: "percent", Value: float64(2.7)},
				}),
			},
		},
	}), o)
}

func TestOrderedMap_UnmarshalJSON_Struct(t *testing.T) {
	t.Parallel()
	var v struct {
		Data *OrderedMap `json:"data"`
	}

	err := json.Unmarshal([]byte(`{ "data": { "x": 1 } }`), &v)
	assert.NoError(t, err)

	value, ok := v.Data.Get("x")
	assert.True(t, ok)
	assert.Equal(t, float64(1), value)
}

// An array of strings decodes to []any like any other JSON array - OrderedMap does not guess a
// field's schema from its runtime content. Callers that know a field is a string array convert
// explicitly via ToStringSlice (see TestToStringSlice). This guards against reintroducing the
// value-dependent []any->[]string coercion removed in PSGO-37: that coercion broke every existing
// `.([]any)` assertion for a string-only array anywhere in a decoded document (orchestrator
// `dependsOn`, `shared_code_row_ids`, JSON-Schema `required`, etc. in keboola-as-code).
func TestOrderedMap_UnmarshalJSON_ArrayOfStrings_StaysAny(t *testing.T) {
	t.Parallel()
	in := `{"rowsSortOrder":["a","b","c"]}`
	m := New()
	assert.NoError(t, json.Unmarshal([]byte(in), &m))
	v, ok := m.Get("rowsSortOrder")
	assert.True(t, ok)
	arr, ok := v.([]any)
	assert.True(t, ok, "expected []any, got %T", v)
	assert.Equal(t, []any{"a", "b", "c"}, arr)

	ss, err := ToStringSlice(v)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, ss)
}

func TestOrderedMap_UnmarshalJSON_NestedArrayOfStrings_StaysAny(t *testing.T) {
	t.Parallel()
	in := `{"arr":[["a","b"],["c"]]}`
	m := New()
	assert.NoError(t, json.Unmarshal([]byte(in), &m))
	v, _ := m.Get("arr")
	outer, ok := v.([]any)
	assert.True(t, ok)
	inner0, ok := outer[0].([]any)
	assert.True(t, ok, "expected []any, got %T", outer[0])
	assert.Equal(t, []any{"a", "b"}, inner0)
	inner1, ok := outer[1].([]any)
	assert.True(t, ok, "expected []any, got %T", outer[1])
	assert.Equal(t, []any{"c"}, inner1)
}

// Regression test: an empty array must stay []any{}, not be silently coerced to []string{}.
// The removed coercion converted every empty array unconditionally, regardless of the field's
// true schema (e.g. an empty "rows": [] array of objects would have become []string{}).
func TestOrderedMap_UnmarshalJSON_EmptyArray_StaysAny(t *testing.T) {
	t.Parallel()
	in := `{"arr":[]}`
	m := New()
	assert.NoError(t, json.Unmarshal([]byte(in), &m))
	v, ok := m.Get("arr")
	assert.True(t, ok)
	arr, ok := v.([]any)
	assert.True(t, ok, "expected []any, got %T", v)
	assert.Empty(t, arr)
}

func TestToStringSlice(t *testing.T) {
	t.Parallel()

	ss, err := ToStringSlice(nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, ss)

	ss, err = ToStringSlice([]any{})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, ss)

	ss, err = ToStringSlice([]any{"a", "b"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, ss)

	// Already-typed []string (e.g. set programmatically via Set) is accepted as-is.
	ss, err = ToStringSlice([]string{"a", "b"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, ss)

	_, err = ToStringSlice("not a slice")
	assert.ErrorContains(t, err, `expected []any or []string, found "string"`)

	_, err = ToStringSlice([]any{"a", 1})
	assert.ErrorContains(t, err, `expected string at index 1, found "int"`)
}
