package deepcopy

import (
	"reflect"
	"strconv"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DeepEqualNotSame checks that a and b values are deep equally, but do not contain any pointer to same value.
func DeepEqualNotSame(t *testing.T, a, b any, path string) {
	t.Helper()

	// Equal
	require.Equal(t, a, b, path)

	// Both nil
	if a == nil || b == nil {
		assert.Nil(t, a)
		assert.Nil(t, b)
		return
	}

	// Same type
	typeA := reflect.TypeOf(a)
	typeB := reflect.TypeOf(b)
	if typeA.String() != typeB.String() {
		require.FailNowf(t, `different types`, `A (%s) and B (%s) have different types`, typeA.String(), typeB.String())
	}

	// But not same (points to different values)
	if typeA.Kind() == reflect.Interface && typeB.Kind() == reflect.Interface {
		require.NotSamef(t, a, b, `%s, path: %s`, typeA.String(), path)
	} else {
		assert.Equal(t, a, b, `%s, path: %s`, typeA.String(), path)
	}

	// Nested fields
	valueA := reflect.ValueOf(a)
	valueB := reflect.ValueOf(b)
	if typeA.Kind() == reflect.Ptr {
		typeA = typeA.Elem()
		valueA = valueA.Elem()
		valueB = valueB.Elem()
	}
	switch typeA.Kind() {
	case reflect.Struct:
		for i := range typeA.NumField() {
			field := typeA.Field(i)
			fieldA := valueA.Field(i)
			fieldB := valueB.Field(i)
			if !fieldA.CanAddr() {
				continue
			} else if !fieldA.CanInterface() {
				// Read unexported fields
				fieldA = reflect.NewAt(field.Type, unsafe.Pointer(fieldA.UnsafeAddr())).Elem() // nolint:gosec // G103: Using unsafe is necessary for testing unexported fields
				fieldB = reflect.NewAt(field.Type, unsafe.Pointer(fieldB.UnsafeAddr())).Elem() // nolint:gosec // G103: Using unsafe is necessary for testing unexported fields
			}

			DeepEqualNotSame(
				t,
				fieldA.Interface(),
				fieldB.Interface(),
				path+`.`+field.Name,
			)
		}
	case reflect.Slice:
		for i := range valueA.Len() {
			DeepEqualNotSame(
				t,
				valueA.Index(i).Interface(),
				valueB.Index(i).Interface(),
				path+`.`+strconv.Itoa(i),
			)

			// Underlying array must be different, check address of the value
			assert.NotSame(t, valueA.Index(i).Addr().Interface(), valueB.Index(i).Addr().Interface(), path+`.`+strconv.Itoa(i))
		}
	case reflect.Map:
		for _, k := range valueA.MapKeys() {
			DeepEqualNotSame(
				t,
				valueA.MapIndex(k).Interface(),
				valueB.MapIndex(k).Interface(),
				path+`.`+k.String(),
			)
		}
	}
}
