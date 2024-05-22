// Package orderedmap is extended version of: https://github.com/iancoleman/orderedmap
//
//	Differences:
//	- Additional methods (GetNested, SetNested, ToMap, ...).
//	- Enhanced JSON decoding: nested map is always pointer (*OrderedMap), this avoids problems with nested values modification.
//	- Added support for deepcopy, see HandleDeepCopy method.
package orderedmap

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/keboola/go-utils/pkg/deepcopy"
)

// Pair a key/value pair.
type Pair struct {
	Key   string
	Value any
}

// ByPair ordered map converted to Pairs.
type ByPair struct {
	Pairs    []*Pair
	LessFunc func(a *Pair, j *Pair) bool
}

func (a ByPair) Len() int           { return len(a.Pairs) }
func (a ByPair) Swap(i, j int)      { a.Pairs[i], a.Pairs[j] = a.Pairs[j], a.Pairs[i] }
func (a ByPair) Less(i, j int) bool { return a.LessFunc(a.Pairs[i], a.Pairs[j]) }

// OrderedMap a map that preserves the order of the keys.
type OrderedMap struct {
	keys   []string
	values map[string]any
}

// VisitCallback callback to visit each nested value in OrderedMap.
type VisitCallback func(path Path, value any, parent any)

// New creates new OrderedMap.
func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]any{}
	return &o
}

// FromPairs creates ordered map from Pairs.
func FromPairs(pairs []Pair) *OrderedMap {
	ordered := New()
	for _, pair := range pairs {
		ordered.Set(pair.Key, pair.Value)
	}
	return ordered
}

// Clone clones ordered map using deepcopy.
func (o *OrderedMap) Clone() *OrderedMap {
	return deepcopy.Copy(o).(*OrderedMap)
}

// HandleDeepCopy implements deepcopy operation.
func (o *OrderedMap) HandleDeepCopy(callback deepcopy.TranslateFn, steps deepcopy.Path, visited deepcopy.VisitedPtrMap) (*OrderedMap, deepcopy.CloneFn) {
	if o == nil {
		return nil, nil
	}
	return New(), func(clone reflect.Value) {
		m := clone.Interface().(*OrderedMap)
		for _, key := range o.Keys() {
			value, _ := o.Get(key)
			keyClone := deepcopy.CopyTranslateSteps(key, callback, steps.Add(MapKeyStep(key)), visited).(string)
			m.Set(keyClone, deepcopy.CopyTranslateSteps(value, callback, steps.Add(MapStep(key)), visited))
		}
	}
}

// ToMap converts OrderedMap to native Go map.
func (o *OrderedMap) ToMap() map[string]any {
	if o == nil {
		return nil
	}

	out := make(map[string]any)
	for k, v := range o.values {
		out[k] = convertToMap(v)
	}

	return out
}

// Get key.
func (o *OrderedMap) Get(key string) (any, bool) {
	val, exists := o.values[key]
	return val, exists
}

// GetOrNil gets key or returns nil if it doesn't exists.
func (o *OrderedMap) GetOrNil(key string) any {
	return o.values[key]
}

// Set key.
func (o *OrderedMap) Set(key string, value any) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// SetNested value defined by path, eg. "parameters.foo[123]".
func (o *OrderedMap) SetNested(path string, value any) error {
	return o.SetNestedPath(PathFromStr(path), value)
}

// SetNestedPath value defined by key, eg. Key{MapStep("parameters), MapStep("foo"), SliceStep(123)}.
func (o *OrderedMap) SetNestedPath(path Path, value any) error {
	if len(path) == 0 {
		return fmt.Errorf(`path cannot be empty`)
	}

	currentKey := make(Path, 0)
	var current any = o

	parentKeys := path.WithoutLast()
	lastKey := path.Last()

	// Get nested map
	for _, key := range parentKeys {
		currentKey = append(currentKey, key)
		switch key := key.(type) {
		case MapStep:
			if m, ok := current.(*OrderedMap); ok {
				if v, found := m.Get(string(key)); found {
					current = v
					continue
				} else {
					newMap := New()
					current = newMap
					m.Set(string(key), newMap)
				}
			} else {
				return fmt.Errorf(`path "%s": expected object found "%T"`, currentKey, current)
			}
		case SliceStep:
			if s, ok := current.([]any); ok {
				if len(s) >= int(key) {
					current = s[key]
					continue
				} else {
					return fmt.Errorf(`path "%s" not found`, currentKey)
				}
			} else {
				return fmt.Errorf(`path "%s": expected array found "%T"`, currentKey.WithoutLast(), current)
			}
		default:
			return fmt.Errorf(`unexpected type "%T"`, key)
		}
	}

	// Set value to map
	if key, ok := lastKey.(MapStep); ok {
		if m, ok := current.(*OrderedMap); ok {
			m.Set(string(key), value)
			return nil
		}
		return fmt.Errorf(`path "%s": expected object found "%T"`, currentKey, current)
	}

	return fmt.Errorf(`path "%s": last key must be MapStep, found "%T"`, path, lastKey)
}

// GetNestedOrNil returns nil if values is not found or an error occurred.
func (o *OrderedMap) GetNestedOrNil(path string) any {
	return o.GetNestedPathOrNil(PathFromStr(path))
}

// GetNestedPathOrNil returns nil if values is not found or an error occurred.
func (o *OrderedMap) GetNestedPathOrNil(path Path) any {
	value, found, err := o.GetNestedPath(path)
	if !found {
		return nil
	} else if err != nil {
		panic(err)
	}
	return value
}

// GetNestedMap returns nested OrderedMap by path as string.
func (o *OrderedMap) GetNestedMap(path string) (m *OrderedMap, found bool, err error) {
	return o.GetNestedPathMap(PathFromStr(path))
}

// GetNestedPathMap returns nested OrderedMap by Path.
func (o *OrderedMap) GetNestedPathMap(path Path) (m *OrderedMap, found bool, err error) {
	value, found, err := o.GetNestedPath(path)
	if !found {
		return nil, false, nil
	} else if err != nil {
		return nil, true, err
	}
	if v, ok := value.(*OrderedMap); ok {
		return v, true, nil
	}
	return nil, true, fmt.Errorf(`path "%s": expected object, found "%T"`, path, value)
}

// GetNested returns nested value by path as string.
func (o *OrderedMap) GetNested(path string) (value any, found bool, err error) {
	return o.GetNestedPath(PathFromStr(path))
}

// GetNestedPath returns nested value by Path.
func (o *OrderedMap) GetNestedPath(path Path) (value any, found bool, err error) {
	if len(path) == 0 {
		return nil, false, fmt.Errorf(`path cannot be empty`)
	}

	currentKey := make(Path, 0)
	var current any = o

	for _, key := range path {
		currentKey = append(currentKey, key)
		switch key := key.(type) {
		case MapStep:
			if m, ok := current.(*OrderedMap); ok {
				if v, found := m.Get(string(key)); found {
					current = v
					continue
				} else {
					return nil, false, fmt.Errorf(`path "%s" not found`, currentKey)
				}
			} else {
				return nil, true, fmt.Errorf(`path "%s": expected object found "%T"`, currentKey.WithoutLast(), current)
			}
		case SliceStep:
			if s, ok := current.([]any); ok {
				if len(s) >= int(key) {
					current = s[key]
					continue
				} else {
					return nil, false, fmt.Errorf(`path "%s" not found`, currentKey)
				}
			} else {
				return nil, true, fmt.Errorf(`path "%s": expected array found "%T"`, currentKey.WithoutLast(), current)
			}
		default:
			return nil, false, fmt.Errorf(`unexpected type "%T"`, key)
		}
	}
	return current, true, nil
}

// VisitAllRecursive calls callback for each nested key in OrderedMap or []any.
func (o *OrderedMap) VisitAllRecursive(callback VisitCallback) {
	visit(Path{}, o, nil, callback)
}

// Delete key from map.
func (o *OrderedMap) Delete(key string) {
	// check key is in use
	if _, ok := o.values[key]; !ok {
		return
	}
	// remove from keys
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	// remove from values
	delete(o.values, key)
}

// Len returns number of keys.
func (o *OrderedMap) Len() int {
	return len(o.keys)
}

// Keys method returns all keys as slice.
func (o *OrderedMap) Keys() []string {
	return o.keys
}

// SortKeys sorts keys using sort func.
func (o *OrderedMap) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.keys)
}

// Sort sorts keys/values using sort func.
func (o *OrderedMap) Sort(lessFunc func(a *Pair, b *Pair) bool) {
	pairs := make([]*Pair, len(o.keys))
	for i, key := range o.keys {
		pairs[i] = &Pair{key, o.values[key]}
	}

	sort.Sort(ByPair{pairs, lessFunc})

	for i, pair := range pairs {
		o.keys[i] = pair.Key
	}
}

func visit(key Path, valueRaw any, parent any, callback VisitCallback) {
	// Call callback for not-root item
	if len(key) != 0 {
		callback(key, valueRaw, parent)
	}

	// Go deep
	switch parent := valueRaw.(type) {
	case *OrderedMap:
		for _, k := range parent.Keys() {
			subValue, _ := parent.Get(k)
			subKey := append(make(Path, 0), key...)
			subKey = append(subKey, MapStep(k))
			visit(subKey, subValue, parent, callback)
		}
	case []any:
		for index, subValue := range parent {
			subKey := append(make(Path, 0), key...)
			subKey = append(subKey, SliceStep(index))
			visit(subKey, subValue, parent, callback)
		}
	}
}

func convertToMap(value any) any {
	switch v := value.(type) {
	case *OrderedMap:
		return v.ToMap()
	case []any:
		mapped := make([]any, 0)
		for _, item := range v {
			mapped = append(mapped, convertToMap(item))
		}
		return mapped
	default:
		return value
	}
}
