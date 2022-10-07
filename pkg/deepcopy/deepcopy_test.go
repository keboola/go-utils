package deepcopy_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	. "github.com/keboola/go-utils/pkg/deepcopy"
	"github.com/keboola/go-utils/pkg/orderedmap"
)

type Values []*Bar

type Foo struct {
	Values Values
}

type Bar struct {
	Key1 string
	Key2 string
	Key3 any // nil interface
}

type UnExportedFields struct {
	key1 string
	key2 string
}

func ExampleCopy() {
	original := map[string]any{"foo": &Bar{Key1: "abc", Key2: "def", Key3: 123}}
	clone := Copy(original).(map[string]any)

	fmt.Printf("pointers are different:  original.foo ==  clone.foo -> %t\n", original["foo"] == clone["foo"])
	fmt.Printf("values are same:        *original.foo == *clone.foo -> %t\n", *original["foo"].(*Bar) == *clone["foo"].(*Bar))
	// Output:
	// pointers are different:  original.foo ==  clone.foo -> false
	// values are same:        *original.foo == *clone.foo -> true
}

func ExampleCopyTranslate() {
	original := map[string]any{"foo": &Bar{Key1: "abc", Key2: "def", Key3: 123}}
	clone := CopyTranslate(original, func(original, clone reflect.Value, path Path) {
		fmt.Printf("Copying: %s\n", path)
		// Uppercase each string
		if clone.Kind() == reflect.String {
			value := strings.ToUpper(clone.Interface().(string))
			clone.Set(reflect.ValueOf(value))
		}
	}).(map[string]any)

	// Dump clone
	fmt.Println()
	fmt.Println("Clone dump:")
	s := spew.NewDefaultConfig()
	s.DisablePointerAddresses = true
	s.Dump(clone)
	// Output:
	// Copying: map[foo].<key>.string
	// Copying: map[foo].interface[*deepcopy_test.Bar].*deepcopy_test.Bar[Key1].string
	// Copying: map[foo].interface[*deepcopy_test.Bar].*deepcopy_test.Bar[Key2].string
	// Copying: map[foo].interface[*deepcopy_test.Bar].*deepcopy_test.Bar[Key3].interface[int].int
	// Copying: map[foo].interface[*deepcopy_test.Bar].*deepcopy_test.Bar[Key3].interface
	// Copying: map[foo].interface[*deepcopy_test.Bar].*struct
	// Copying: map[foo].interface[*deepcopy_test.Bar].ptr
	// Copying: map[foo].interface
	// Copying: map
	//
	// Clone dump:
	// (map[string]interface {}) (len=1) {
	//  (string) (len=3) "FOO": (*deepcopy_test.Bar)({
	//   Key1: (string) (len=3) "ABC",
	//   Key2: (string) (len=3) "DEF",
	//   Key3: (int) 123
	//  })
	// }
}

func TestCopy(t *testing.T) {
	t.Parallel()
	original := inputValue()
	clone := Copy(original)
	assert.Equal(t, original, clone)
	assert.NotSame(t, original, clone)
	DeepEqualNotSame(t, original, clone, "")
}

func TestCopyWithTranslate(t *testing.T) {
	t.Parallel()
	original := inputValue()
	clone := CopyTranslate(original, func(_, clone reflect.Value, _ Path) {
		// Modify all strings
		if clone.Kind() == reflect.String {
			clone.Set(reflect.ValueOf(clone.Interface().(string) + "_modified"))
		}
	})
	assert.Equal(t, expectedValueModifiedStrings(), clone)
}

func TestCopyWithTranslatePath(t *testing.T) {
	t.Parallel()
	original := inputValue()
	clone := CopyTranslate(original, func(_, clone reflect.Value, path Path) {
		// Modify all strings
		if clone.Kind() == reflect.String && !strings.Contains(path.String(), ".<key>") {
			clone.Set(reflect.ValueOf(path.String()))
		}
	})
	assert.Equal(t, expectedValueSteps(), clone)
}

func TestCopyCycle(t *testing.T) {
	t.Parallel()
	m := orderedmap.New()
	m.Set("key", m)
	c := Copy(m).(*orderedmap.OrderedMap)
	ck, _ := c.Get("key")
	assert.NotSame(t, m, c)
	assert.NotSame(t, m, ck)
	assert.Equal(t, m, c)
	assert.Equal(t, m, ck)
}

func TestCopyUnexportedFields(t *testing.T) {
	t.Parallel()
	m := orderedmap.New()
	m.Set("key", &UnExportedFields{key1: "a", key2: "b"})
	expected := `
deepcopy found unexported field:
  path: *orderedmap.OrderedMap[key].*deepcopy_test.UnExportedFields[key1]
  value: deepcopy_test.UnExportedFields{key1:"a", key2:"b"}
`
	assert.PanicsWithError(t, strings.TrimSpace(expected), func() {
		Copy(m)
	})
}

func inputValue() any {
	m := orderedmap.New()
	m.Set("foo", &Foo{
		Values: []*Bar{
			{
				Key1: "value1",
				Key2: "value2",
			},
			{
				Key1: "value3",
				Key2: "value4",
			},
		},
	})
	m.Set("bar", Bar{
		Key1: "value1",
		Key2: "value2",
	})
	m.Set("[]empty", []any(nil))
	m.Set("[]bar", []any{
		Bar{
			Key1: "value1",
			Key2: "value2",
		},
		Bar{
			Key1: "value1",
			Key2: "value2",
		},
	})

	subMap := orderedmap.New()
	subMap.Set("key1", 123)
	subMap.Set("key2", 456)
	m.Set("subMap", subMap)

	m.Set("nativeMap", map[string]int{
		"foo": 123,
	})

	return m
}

// expectedValueModifiedStrings - to each string is added suffix "_modified".
func expectedValueModifiedStrings() any {
	m := orderedmap.New()
	m.Set("foo_modified", &Foo{
		Values: []*Bar{
			{
				Key1: "value1_modified",
				Key2: "value2_modified",
			},
			{
				Key1: "value3_modified",
				Key2: "value4_modified",
			},
		},
	})
	m.Set("bar_modified", Bar{
		Key1: "value1_modified",
		Key2: "value2_modified",
	})
	m.Set("[]empty_modified", []any(nil))
	m.Set("[]bar_modified", []any{
		Bar{
			Key1: "value1_modified",
			Key2: "value2_modified",
		},
		Bar{
			Key1: "value1_modified",
			Key2: "value2_modified",
		},
	})

	subMap := orderedmap.New()
	subMap.Set("key1_modified", 123)
	subMap.Set("key2_modified", 456)
	m.Set("subMap_modified", subMap)

	m.Set("nativeMap_modified", map[string]int{
		"foo_modified": 123,
	})

	return m
}

// expectedValueSteps - each string value is replaced by serialized path to the value.
func expectedValueSteps() any {
	m := orderedmap.New()
	m.Set("foo", &Foo{
		Values: []*Bar{
			{
				Key1: "*orderedmap.OrderedMap[foo].*deepcopy_test.Foo[Values].slice[0].*deepcopy_test.Bar[Key1].string",
				Key2: "*orderedmap.OrderedMap[foo].*deepcopy_test.Foo[Values].slice[0].*deepcopy_test.Bar[Key2].string",
			},
			{
				Key1: "*orderedmap.OrderedMap[foo].*deepcopy_test.Foo[Values].slice[1].*deepcopy_test.Bar[Key1].string",
				Key2: "*orderedmap.OrderedMap[foo].*deepcopy_test.Foo[Values].slice[1].*deepcopy_test.Bar[Key2].string",
			},
		},
	})
	m.Set("bar", Bar{
		Key1: "*orderedmap.OrderedMap[bar].deepcopy_test.Bar[Key1].string",
		Key2: "*orderedmap.OrderedMap[bar].deepcopy_test.Bar[Key2].string",
	})
	m.Set("[]empty", []any(nil))
	m.Set("[]bar", []any{
		Bar{
			Key1: "*orderedmap.OrderedMap[[]bar].slice[0].interface[deepcopy_test.Bar].deepcopy_test.Bar[Key1].string",
			Key2: "*orderedmap.OrderedMap[[]bar].slice[0].interface[deepcopy_test.Bar].deepcopy_test.Bar[Key2].string",
		},
		Bar{
			Key1: "*orderedmap.OrderedMap[[]bar].slice[1].interface[deepcopy_test.Bar].deepcopy_test.Bar[Key1].string",
			Key2: "*orderedmap.OrderedMap[[]bar].slice[1].interface[deepcopy_test.Bar].deepcopy_test.Bar[Key2].string",
		},
	})

	subMap := orderedmap.New()
	subMap.Set("key1", 123)
	subMap.Set("key2", 456)
	m.Set("subMap", subMap)

	m.Set("nativeMap", map[string]int{
		"foo": 123,
	})

	return m
}
