package orderedmap

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestOrderedMap_MarshalYAML(t *testing.T) {
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
	o.Set("a", &yaml.Node{Kind: yaml.ScalarNode, Value: "2", LineComment: "line comment"})
	o.Set("b", 3)
	// slice
	o.Set("slice", []any{
		&yaml.Node{Kind: yaml.ScalarNode, Value: "1", Style: yaml.DoubleQuotedStyle, HeadComment: "head comment"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "1", LineComment: "line comment"},
	})
	// orderedmap
	v := New()
	v.Set("e", &yaml.Node{Kind: yaml.ScalarNode, Value: "1", LineComment: "line comment"})
	v.Set("a", &yaml.Node{Kind: yaml.ScalarNode, Value: "2", HeadComment: "head comment"})
	o.Set("orderedmap", v)
	// escape key
	o.Set("test\n\r\t\\\"ing", 9)

	// result
	expected := `
number: 4
string: x
specialstring: \.<>[]{}_-
z: 1
a: 2 # line comment
b: 3
slice:
  # head comment
  - "1"
  - 1 # line comment
orderedmap:
  e: 1 # line comment
  # head comment
  a: 2
? "test\n\r\t\\\"ing"
: 9
`
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	assert.NoError(t, encoder.Encode(o))
	assert.Equal(t, strings.TrimLeft(expected, "\n"), out.String())
}

func TestOrderedMap_MarshalYAML_Blank(t *testing.T) {
	t.Parallel()
	o := New()

	// blank map
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	assert.NoError(t, encoder.Encode(o))
	assert.Equal(t, "{}\n", out.String())
}

func TestOrderedMap_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	in := `
number: 4
string: x
z: 1
a: should not break with unclosed { character in value
b: 3
slice:
- '1'
- 1
orderedmap:
  e: 1
  a { nested key with brace: with a }}}} }} {{{ brace value
  after:
    link: test {{{ with even deeper nested braces }
test"ing: 9
after: 1
multitype_array:
- test
- 1
- map: obj
  it: 5
  ":colon in key": 'colon: in value'
- - inner: map
should not break with { character in key: 1
`
	o := New()
	err := yaml.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "number",
			Value: 4,
		},
		{
			Key:   "string",
			Value: "x",
		},
		{
			Key:   "z",
			Value: 1,
		},
		{
			Key:   "a",
			Value: "should not break with unclosed { character in value",
		},
		{
			Key:   "b",
			Value: 3,
		},
		{
			Key: "slice",
			Value: []any{
				"1",
				1,
			},
		},
		{
			Key: "orderedmap",
			Value: FromPairs([]Pair{
				{
					Key:   "e",
					Value: 1,
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
			Value: 9,
		},
		{
			Key:   "after",
			Value: 1,
		},
		{
			Key: "multitype_array",
			Value: []any{
				"test",
				1,
				FromPairs([]Pair{
					{
						Key:   "map",
						Value: "obj",
					},
					{
						Key:   "it",
						Value: 5,
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
			Value: 1,
		},
	}), o)
}

func TestOrderedMap_UnmarshalYAML_DuplicateKeys(t *testing.T) {
	t.Parallel()
	in := `
a:
- {}
- []
b:
  x:
  - 1
c: x
d:
  x: 1
b:
- x: []
c: 1
d:
  y: 2
e:
- x: 1
e:
- []
e:
- z: 2
a: {}
b:
- - 1
`

	o := New()
	err := yaml.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "c",
			Value: 1,
		},
		{
			Key: "d",
			Value: FromPairs([]Pair{
				{
					Key:   "y",
					Value: 2,
				},
			}),
		},
		{
			Key: "e",
			Value: []any{FromPairs([]Pair{
				{
					Key:   "z",
					Value: 2,
				},
			})},
		},
		{
			Key:   "a",
			Value: New(),
		},
		{
			Key:   "b",
			Value: []any{[]any{1}},
		},
	}), o)
}

func TestOrderedMap_UnmarshalYAML_SpecialChars(t *testing.T) {
	t.Parallel()
	in := `
" \u0041\n\r\t\\\\\\\\\\\\ ":
  "\\\\\\": "\\\\\"\\"
"\\": " \\\\ test "
"\n": "\r"
`

	o := New()
	err := yaml.Unmarshal([]byte(in), &o)
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

func TestOrderedMap_UnmarshalYAML_ArrayOfMaps(t *testing.T) {
	t.Parallel()
	in := `
name: test
percent: 6
breakdown:
  - name: a
    percent: 0.9
  - name: b
    percent: 0.9
  - name: d
    percent: 0.4
  - name: e
    percent: 2.7

`
	o := New()
	err := yaml.Unmarshal([]byte(in), &o)
	assert.NoError(t, err)
	assert.Equal(t, FromPairs([]Pair{
		{
			Key:   "name",
			Value: "test",
		},
		{
			Key:   "percent",
			Value: 6,
		},
		{
			Key: "breakdown",
			Value: []any{
				FromPairs([]Pair{
					{Key: "name", Value: "a"},
					{Key: "percent", Value: 0.9},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "b"},
					{Key: "percent", Value: 0.9},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "d"},
					{Key: "percent", Value: 0.4},
				}),
				FromPairs([]Pair{
					{Key: "name", Value: "e"},
					{Key: "percent", Value: 2.7},
				}),
			},
		},
	}), o)
}

func TestOrderedMap_UnmarshalYAML_Struct(t *testing.T) {
	t.Parallel()
	var v struct {
		Data *OrderedMap `yaml:"data"`
	}

	err := yaml.Unmarshal([]byte("data:\n  x: 1\n"), &v)
	assert.NoError(t, err)

	value, ok := v.Data.Get("x")
	assert.True(t, ok)
	assert.Equal(t, 1, value)
}

func TestOrderedMap_UnmarshalYAML_Text(t *testing.T) {
	t.Parallel()
	o := New()
	err := yaml.Unmarshal([]byte("some text"), o)
	assert.Error(t, err)
	assert.Equal(t, "cannot unmarshal !!str `some text` into orderedmap", err.Error())
}
