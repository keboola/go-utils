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
	in := `{ " \u0041\n\r\t\\\\\\\\\\\\ "  : { "\\\\\\" : "\\\\\"\\" }, "\\":  " \\\\ test ", "\n": "\r" }`

	o := New()
	err := json.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key: " \u0041\n\r\t\\\\\\\\\\\\ ",
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
