package deepcopy

import (
	"fmt"
	"reflect"
	"strings"
)

// Path to a nested value.
type Path []fmt.Stringer

// Add step.
func (s Path) Add(step fmt.Stringer) Path {
	newIndex := len(s)
	out := make(Path, newIndex+1)
	copy(out, s)
	out[newIndex] = step
	return out
}

func (s Path) String() string {
	var out []string
	for _, item := range s {
		out = append(out, item.String())
	}
	str := strings.Join(out, `.`)
	str = strings.ReplaceAll(str, `*.`, `*`)
	str = strings.ReplaceAll(str, `.[`, `[`)
	return str
}

// TypeStep - type information.
type TypeStep struct {
	CurrentType string
}

func (v TypeStep) String() string {
	return v.CurrentType
}

// PointerStep - pointer dereference.
type PointerStep struct{}

func (v PointerStep) String() string {
	return "*"
}

// InterfaceStep - interface dereference.
type InterfaceStep struct {
	TargetType reflect.Type
}

func (v InterfaceStep) String() string {
	return fmt.Sprintf("interface[%s]", v.TargetType)
}

// StructFieldStep - field in a struct.
type StructFieldStep struct {
	CurrentType reflect.Type
	Field       string
}

func (v StructFieldStep) String() string {
	return fmt.Sprintf("%s[%s]", v.CurrentType, v.Field)
}

// SliceIndexStep - index in a slice.
type SliceIndexStep struct {
	Index int
}

func (v SliceIndexStep) String() string {
	return fmt.Sprintf("slice[%d]", v.Index)
}

// MapKeyStep - key in a map.
type MapKeyStep struct {
	Key any
}

func (v MapKeyStep) String() string {
	return fmt.Sprintf("map[%v]", v.Key)
}

// MapKeyValueStep - value in a map.
type MapKeyValueStep struct {
	Key any
}

func (v MapKeyValueStep) String() string {
	return fmt.Sprintf("map[%v].<key>", v.Key)
}
