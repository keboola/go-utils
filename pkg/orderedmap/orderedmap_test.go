// nolint: ifshort
package orderedmap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderedMap(t *testing.T) {
	t.Parallel()
	o := New()
	// number
	o.Set("number", 3)
	v, _ := o.Get("number")
	if v.(int) != 3 {
		t.Error("Set number")
	}
	// string
	o.Set("string", "x")
	v, _ = o.Get("string")
	if v.(string) != "x" {
		t.Error("Set string")
	}
	// string slice
	o.Set("strings", []string{
		"t",
		"u",
	})
	v, _ = o.Get("strings")
	if v.([]string)[0] != "t" {
		t.Error("Set strings first index")
	}
	if v.([]string)[1] != "u" {
		t.Error("Set strings second index")
	}
	// mixed slice
	o.Set("mixed", []any{
		1,
		"1",
	})
	v, _ = o.Get("mixed")
	if v.([]any)[0].(int) != 1 {
		t.Error("Set mixed int")
	}
	if v.([]any)[1].(string) != "1" {
		t.Error("Set mixed string")
	}
	// overriding existing key
	o.Set("number", 4)
	v, _ = o.Get("number")
	if v.(int) != 4 {
		t.Error("Override existing key")
	}
	// Keys method
	keys := o.Keys()
	expectedKeys := []string{
		"number",
		"string",
		"strings",
		"mixed",
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	for i, key := range expectedKeys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	// delete
	o.Delete("strings")
	o.Delete("not a key being used")
	if len(o.Keys()) != 3 {
		t.Error("Delete method")
	}
	if _, ok := o.Get("strings"); ok {
		t.Error("Delete did not remove 'strings' key")
	}
}

func TestOrderedMap_SortKeys(t *testing.T) {
	t.Parallel()
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	assert.NoError(t, json.Unmarshal([]byte(s), &o))

	o.SortKeys(sort.Strings)

	// Check the root keys
	expectedKeys := []string{
		"a",
		"b",
		"c",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("SortKeys root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

func TestOrderedMap_Sort(t *testing.T) {
	t.Parallel()
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	assert.NoError(t, json.Unmarshal([]byte(s), &o))
	o.Sort(func(a *Pair, b *Pair) bool {
		return a.Value.(float64) > b.Value.(float64)
	})

	// Check the root keys
	expectedKeys := []string{
		"c",
		"b",
		"a",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Sort root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

// https://github.com/iancoleman/orderedmap/issues/11
func TestOrderedMap_empty_array(t *testing.T) {
	t.Parallel()
	srcStr := `{"x":[]}`
	src := []byte(srcStr)
	om := New()
	assert.NoError(t, json.Unmarshal(src, om))
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty array does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}

// Inspired by
// https://github.com/iancoleman/orderedmap/issues/11
// but using empty maps instead of empty slices.
func TestOrderedMap_empty_map(t *testing.T) {
	t.Parallel()
	srcStr := `{"x":{}}`
	src := []byte(srcStr)
	om := New()
	assert.NoError(t, json.Unmarshal(src, om))
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty map does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}

func TestOrderedMap_Clone(t *testing.T) {
	t.Parallel()
	root := New()
	nested := New()
	nested.Set(`key`, `value`)
	root.Set(`nested`, nested)

	rootClone := root.Clone()
	assert.NotSame(t, root, rootClone)
	assert.Equal(t, root, rootClone)

	nestedClone, found := rootClone.Get(`nested`)
	assert.True(t, found)
	assert.NotSame(t, nested, nestedClone)
	assert.Equal(t, nested, nestedClone)
}

func TestOrderedMap_ToMap(t *testing.T) {
	t.Parallel()
	root := New()
	nested := New()
	nested.Set(`key`, `value`)
	root.Set(`nested`, nested)

	assert.Equal(t, map[string]any{
		`nested`: map[string]any{
			`key`: `value`,
		},
	}, root.ToMap())
}

func TestOrderedMapGetNested(t *testing.T) {
	t.Parallel()
	root := New()
	nested := New()
	nested.Set(`key`, `value`)
	nested.Set(`slice`, []any{1, 2, 3})
	root.Set(`nested`, nested)
	root.Set(`slice`, []any{1, 2, 3})

	// Missing root map key
	value, found, err := root.GetNested(`foo`)
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "foo" not found`, err.Error())
	value = root.GetNestedOrNil(`foo`)
	assert.Nil(t, value)
	value, found, err = root.GetNestedPath(Path{MapStep(`foo`)})
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "foo" not found`, err.Error())
	value = root.GetNestedPathOrNil(Path{MapStep(`foo`)})
	assert.Nil(t, value)

	// Missing root slice key
	value, found, err = root.GetNested(`foo[123]`)
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "foo" not found`, err.Error())
	value = root.GetNestedOrNil(`foo[123]`)
	assert.Nil(t, value)
	value, found, err = root.GetNestedPath(Path{MapStep(`foo`), SliceStep(123)})
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "foo" not found`, err.Error())
	value = root.GetNestedPathOrNil(Path{MapStep(`foo`), SliceStep(123)})
	assert.Nil(t, value)

	// Missing nested map key
	value, found, err = root.GetNested(`nested.foo`)
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.foo" not found`, err.Error())
	value = root.GetNestedOrNil(`nested.foo`)
	assert.Nil(t, value)
	value, found, err = root.GetNestedPath(Path{MapStep(`nested`), MapStep(`foo`)})
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.foo" not found`, err.Error())
	value = root.GetNestedPathOrNil(Path{MapStep(`nested`), MapStep(`foo`)})
	assert.Nil(t, value)

	// Missing nested slice key
	value, found, err = root.GetNested(`nested.slice[3]`)
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.slice[3]" not found`, err.Error())
	value = root.GetNestedOrNil(`nested.slice[3]`)
	assert.Nil(t, value)
	value, found, err = root.GetNestedPath(Path{MapStep(`nested`), MapStep(`slice`), SliceStep(3)})
	assert.Nil(t, value)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.slice[3]" not found`, err.Error())
	value = root.GetNestedPathOrNil(Path{MapStep(`nested`), MapStep(`slice`), SliceStep(3)})
	assert.Nil(t, value)

	// Get nested map - not found
	value, found, err = root.GetNestedMap(`nested.foo`)
	assert.Nil(t, value)
	assert.False(t, found)
	assert.NoError(t, err)
	value, found, err = root.GetNestedPathMap(Path{MapStep(`nested`), MapStep(`foo`)})
	assert.Nil(t, value)
	assert.False(t, found)
	assert.NoError(t, err)

	// Get nested map - found
	value, found, err = root.GetNestedMap(`nested`)
	assert.Equal(t, nested, value)
	assert.True(t, found)
	assert.NoError(t, err)
	value, found, err = root.GetNestedPathMap(Path{MapStep(`nested`)})
	assert.Equal(t, nested, value)
	assert.True(t, found)
	assert.NoError(t, err)

	// Get nested map - invalid type
	value, found, err = root.GetNestedMap(`nested.key`)
	assert.Nil(t, value)
	assert.True(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.key": expected object, found "string"`, err.Error())
	value, found, err = root.GetNestedPathMap(Path{MapStep(`nested`), MapStep(`key`)})
	assert.Nil(t, value)
	assert.True(t, found)
	assert.Error(t, err)
	assert.Equal(t, `path "nested.key": expected object, found "string"`, err.Error())
}

func TestOrderedMapSetNested(t *testing.T) {
	t.Parallel()
	root := New()

	// Set top level key
	assert.NoError(t, root.SetNested(`foo1`, `bar1`))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`foo2`)}, `bar2`))

	// Set nested key
	assert.NoError(t, root.SetNested(`nested`, New()))
	assert.NoError(t, root.SetNested(`nested.foo3`, `bar3`))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`nested`), MapStep(`foo4`)}, `bar4`))

	// Set nested - parent not found
	assert.NoError(t, root.SetNested(`nested.missing.key`, `value`))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`nested`), MapStep(`missing`), MapStep(`key`)}, `value`))

	// Set nested in slice
	assert.NoError(t, root.SetNested(`slice`, []any{New()}))
	assert.NoError(t, root.SetNested(`slice[0].foo`, 4))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`slice`), SliceStep(0), MapStep(`foo`)}, 4))

	// Set slice value
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`slice2`), SliceStep(0)}, 4))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`slice2`), SliceStep(1)}, 5))
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`slice2`), SliceStep(5)}, 6))

	// Set value in nested slices
	assert.NoError(t, root.SetNestedPath(Path{MapStep(`slice3`), SliceStep(1), SliceStep(1), SliceStep(1)}, 7))

	// Set nested in slice
	assert.NoError(t, root.SetNested(`slice[2].foo[1]`, 5))

	// Set nested - invalid type
	assert.NoError(t, root.SetNested(`str`, `value`))
	err := root.SetNested(`str.key`, `value`)
	assert.Error(t, err)
	assert.Equal(t, `path "str.key": expected object found "string"`, err.Error())
	err = root.SetNestedPath(Path{MapStep(`str`), MapStep(`key`)}, `value`)
	assert.Error(t, err)
	assert.Equal(t, `path "str.key": expected object found "string"`, err.Error())

	// Invalid: first step is SliceStep
	err = root.SetNestedPath(Path{SliceStep(0), SliceStep(1)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `first key must be MapStep, found "orderedmap.SliceStep"`, err.Error())

	// Invalid: negative slice step
	err = root.SetNestedPath(Path{MapStep("test"), SliceStep(-1)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "test[-1]": array key can't be negative`, err.Error())
	err = root.SetNestedPath(Path{MapStep("test"), SliceStep(-1), MapStep("test")}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "test[-1]": array key can't be negative`, err.Error())

	// Invalid: unknown step type
	err = root.SetNestedPath(Path{MapStep("test"), MapKeyStep("test")}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "test[test].<key>": last key must be MapStep of SliceStep, found "orderedmap.MapKeyStep"`, err.Error())
	err = root.SetNestedPath(Path{MapStep("test"), MapKeyStep("test"), MapStep("test")}, 1)
	assert.Error(t, err)
	assert.Equal(t, `unexpected type "orderedmap.MapKeyStep"`, err.Error())

	// Invalid: empty path
	err = root.SetNestedPath(Path{}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path cannot be empty`, err.Error())

	// Invalid: using MapStep on slice
	err = root.SetNestedPath(Path{MapStep(`slice`), MapStep(`test`), SliceStep(1)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "slice.test": expected object found "[]interface {}"`, err.Error())
	err = root.SetNestedPath(Path{MapStep(`slice`), MapStep(`test`)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "slice.test": expected object found "[]interface {}"`, err.Error())

	// Invalid: using SliceStep on map
	err = root.SetNestedPath(Path{MapStep(`nested`), SliceStep(0), SliceStep(1)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "nested[0]": expected array found "*orderedmap.OrderedMap"`, err.Error())
	err = root.SetNestedPath(Path{MapStep(`nested`), SliceStep(0)}, 1)
	assert.Error(t, err)
	assert.Equal(t, `path "nested[0]": expected array found "*orderedmap.OrderedMap"`, err.Error())

	expected := `
{
  "foo1": "bar1",
  "foo2": "bar2",
  "nested": {
    "foo3": "bar3",
    "foo4": "bar4",
    "missing": {
      "key": "value"
    }
  },
  "slice": [
    {
      "foo": 4
    },
    null,
    {
      "foo": [
        null,
        5
      ]
    }
  ],
  "slice2": [
    4,
    5,
    null,
    null,
    null,
    6
  ],
  "slice3": [
    null,
    [
      null,
      [
        null,
        7
      ]
    ]
  ],
  "str": "value",
  "test": []
}
`
	jsonBytes, err := json.MarshalIndent(root, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(expected), string(jsonBytes))
}

func TestFromPairs(t *testing.T) {
	t.Parallel()
	m := FromPairs([]Pair{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: 123},
	})

	expected := `
{
  "key1": "value1",
  "key2": 123
}
`
	jsonBytes, err := json.MarshalIndent(m, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(expected), string(jsonBytes))
}

func TestOrderedMap_VisitAllRecursive(t *testing.T) {
	t.Parallel()
	input := `
{
    "foo1": "bar1",
    "foo2": "bar2",
    "nested1": {
        "foo3": "bar3",
        "foo4": "bar4",
        "nested2": {
            "key": "value"
        },
        "slice": [
            123,
            "abc",
            {
                "nested3": {
                    "foo5": "bar5"
                }
            },
            {
                "subSlice": [
                    456,
                    "def",
                    {
                        "nested4": {
                            "foo6": "bar6"
                        }
                    }
                ]
            }
        ]
    },
    "str": "value"
}
`

	expected := `
path=foo1, parent=*orderedmap.OrderedMap, value=string
path=foo2, parent=*orderedmap.OrderedMap, value=string
path=nested1, parent=*orderedmap.OrderedMap, value=*orderedmap.OrderedMap
path=nested1.foo3, parent=*orderedmap.OrderedMap, value=string
path=nested1.foo4, parent=*orderedmap.OrderedMap, value=string
path=nested1.nested2, parent=*orderedmap.OrderedMap, value=*orderedmap.OrderedMap
path=nested1.nested2.key, parent=*orderedmap.OrderedMap, value=string
path=nested1.slice, parent=*orderedmap.OrderedMap, value=[]interface {}
path=nested1.slice[0], parent=[]interface {}, value=float64
path=nested1.slice[1], parent=[]interface {}, value=string
path=nested1.slice[2], parent=[]interface {}, value=*orderedmap.OrderedMap
path=nested1.slice[2].nested3, parent=*orderedmap.OrderedMap, value=*orderedmap.OrderedMap
path=nested1.slice[2].nested3.foo5, parent=*orderedmap.OrderedMap, value=string
path=nested1.slice[3], parent=[]interface {}, value=*orderedmap.OrderedMap
path=nested1.slice[3].subSlice, parent=*orderedmap.OrderedMap, value=[]interface {}
path=nested1.slice[3].subSlice[0], parent=[]interface {}, value=float64
path=nested1.slice[3].subSlice[1], parent=[]interface {}, value=string
path=nested1.slice[3].subSlice[2], parent=[]interface {}, value=*orderedmap.OrderedMap
path=nested1.slice[3].subSlice[2].nested4, parent=*orderedmap.OrderedMap, value=*orderedmap.OrderedMap
path=nested1.slice[3].subSlice[2].nested4.foo6, parent=*orderedmap.OrderedMap, value=string
path=str, parent=*orderedmap.OrderedMap, value=string
`

	m := New()
	assert.NoError(t, json.Unmarshal([]byte(input), m))

	var visited []string
	m.VisitAllRecursive(func(path Path, value any, parent any) {
		visited = append(visited, fmt.Sprintf(`path=%s, parent=%T, value=%T`, path, parent, value))
	})
	assert.Equal(t, strings.TrimSpace(expected), strings.Join(visited, "\n"))
}
