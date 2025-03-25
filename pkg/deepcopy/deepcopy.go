// Package deepcopy implements deep copy and deep translate of a value.
//
// It is extended version of https://gist.github.com/hvoecking/10772475.
//
// TranslateFn can be used to modify value on copying.
//
// CustomDeepCopyMethod can be defined on a type, for example, to copy unexported fields.
// See "github.com/keboola/go-utils/pkg/orderedmap" package for example of CustomDeepCopyMethod.
package deepcopy

import (
	"fmt"
	"reflect"
)

// CustomDeepCopyMethod is name of the method that handles deep copy for the type.
const CustomDeepCopyMethod = "HandleDeepCopy"

// TranslateFn is custom translate function to modify values on copying.
type TranslateFn func(original, clone reflect.Value, path Path)

// CloneFn is custom implementation of deepcopy for a type, it is returned from CustomDeepCopyMethod.
type CloneFn func(clone reflect.Value)

// VisitedPtrMap maps pointer from original value to cloned value.
// Example: If, in original value A, is 3x a pointer that point to the value B.
// Then, in the cloned value AC, there will be 3x pointer to the cloned value BC.
type VisitedPtrMap map[uintptr]*reflect.Value

// Copy makes deep copy of the value.
func Copy(value any) any {
	return CopyTranslate(value, nil)
}

// CopyTranslate makes deep copy of the value, each value is translated by TranslateFn.
func CopyTranslate(value any, callback TranslateFn) any {
	return CopyTranslateSteps(value, callback, Path{}, make(VisitedPtrMap))
}

// CopyTranslateSteps makes deep copy of the value, each value is translated by TranslateFn.
// VisitedPtrMap allows you to connect copy to another copy operation and reuse pointers.
func CopyTranslateSteps(value any, callback TranslateFn, path Path, visited VisitedPtrMap) any {
	if value == nil {
		return nil
	}

	// Wrap the original in a reflect.Value
	original := reflect.ValueOf(value)
	clone := reflect.New(original.Type()).Elem()
	translateRecursive(clone, original, callback, path, visited)

	// Remove the reflection wrapper
	return clone.Interface()
}

func translateRecursive(clone, original reflect.Value, callback TranslateFn, path Path, visitedPtr VisitedPtrMap) {
	originalType := original.Type()
	cloneMethod, cloneMethodFound := originalType.MethodByName(CustomDeepCopyMethod)
	kind := original.Kind()

	// Process if multiple pointers point to the same value
	if kind == reflect.Ptr && !original.IsNil() {
		ptr := original.Pointer()
		// Cloned value found, return
		if v, found := visitedPtr[ptr]; found {
			clone.Set(*v)
			return
		}
		// Cloned value not found, continue
		visitedPtr[ptr] = &clone
	}

	switch {
	// Use CustomDeepCopyMethod method if is present
	case cloneMethodFound && cloneMethod.Type.Out(0).String() == originalType.String():
		values := original.MethodByName(CustomDeepCopyMethod).Call([]reflect.Value{
			reflect.ValueOf(callback),
			reflect.ValueOf(path.Add(TypeStep{CurrentType: originalType.String()})),
			reflect.ValueOf(visitedPtr),
		})
		if len(values) != 2 {
			panic(fmt.Errorf(`expected two return value from %s.%s, got %d`, cloneMethod.PkgPath, cloneMethod.Name, len(values)))
		}
		clone.Set(values[0])
		if values[1].IsValid() {
			if fn, ok := values[1].Interface().(CloneFn); !ok {
				panic(fmt.Errorf(`second return value from %s.%s must be "CloneFn", got %d`, cloneMethod.PkgPath, cloneMethod.Name, len(values)))
			} else if fn != nil {
				fn(clone)
			}
		}
	// If it is a pointer we need to unwrap and call once again
	case kind == reflect.Ptr:
		// Check if the pointer is nil
		originalValue := original.Elem()
		if originalValue.IsValid() {
			// Allocate a new object and set the pointer to it
			clone.Set(reflect.New(originalValue.Type()))
			// Unwrap the newly created pointer
			path := path.Add(PointerStep{})
			translateRecursive(clone.Elem(), originalValue, callback, path, visitedPtr)
		}

	// If it is an interface (which is very similar to a pointer), do basically the
	// same as for the pointer. Though a pointer is not the same as an interface so
	// note that we have to call Elem() after creating a new object because otherwise
	// we would end up with an actual pointer
	case kind == reflect.Interface:
		// Get rid of the wrapping interface
		originalValue := original.Elem()
		// Check if the pointer is nil
		if originalValue.IsValid() {
			// Create a new object. Now new gives us a pointer, but we want the value it
			// points to, so we have to call Elem() to unwrap it
			t := originalValue.Type()
			cloneValue := reflect.New(t).Elem()
			path := path.Add(InterfaceStep{TargetType: t})
			translateRecursive(cloneValue, originalValue, callback, path, visitedPtr)
			clone.Set(cloneValue)
		}

	// If it is a struct we translate each field
	case kind == reflect.Struct:
		t := originalType
		for i := range original.NumField() {
			path := path.Add(StructFieldStep{CurrentType: originalType, Field: t.Field(i).Name})
			cloneField := clone.Field(i)
			if !cloneField.CanSet() {
				panic(fmt.Errorf("deepcopy found unexported field:\n  path: %s\n  value: %#v", path.String(), original.Interface()))
			}
			translateRecursive(cloneField, original.Field(i), callback, path, visitedPtr)
		}

	// If it is a slice we create a new slice and translate each element
	case kind == reflect.Slice:
		if !original.IsNil() {
			clone.Set(reflect.MakeSlice(originalType, original.Len(), original.Cap()))
			for i := range original.Len() {
				path := path.Add(SliceIndexStep{Index: i})
				translateRecursive(clone.Index(i), original.Index(i), callback, path, visitedPtr)
			}
		}

	// If it is a map we create a new map and translate each value
	case kind == reflect.Map:
		if !original.IsNil() {
			clone.Set(reflect.MakeMap(originalType))
			for _, originalKey := range original.MapKeys() {
				// Clone key
				cloneKey := reflect.New(originalKey.Type()).Elem()
				keySteps := path.Add(MapKeyValueStep{Key: originalKey.Interface()})
				translateRecursive(cloneKey, originalKey, callback, keySteps, visitedPtr)

				// New gives us a pointer, but again we want the value
				originalValue := original.MapIndex(originalKey)
				cloneValue := reflect.New(originalValue.Type()).Elem()
				path := path.Add(MapKeyStep{Key: originalKey.Interface()})
				translateRecursive(cloneValue, originalValue, callback, path, visitedPtr)

				clone.SetMapIndex(cloneKey, cloneValue)
			}
		}

	// And everything else will simply be taken from the original
	default:
		clone.Set(original)
	}

	// Custom modifications
	if callback != nil {
		callback(original, clone, path.Add(TypeStep{CurrentType: kind.String()}))
	}
}
