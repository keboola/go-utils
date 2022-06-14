package orderedmap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var sliceStepRegexp = regexp.MustCompile(`^(\d+)\]$`)

// Path to a nested value in the OrderedMap.
type Path []Step

// Step of Path.
type Step interface {
	String() string
}

// MapStep represents a map key value.
type MapStep string

// MapKeyStep represents a map key, used with deepcopy.
type MapKeyStep string

// SliceStep represents a slice index.
type SliceStep int

// PathFromStr converts string to Path.
func PathFromStr(str string) Path {
	parts := strings.FieldsFunc(str, func(r rune) bool {
		return r == '.' || r == '['
	})

	out := make(Path, 0)
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		// Is slice step? eg. [123]
		matches := sliceStepRegexp.FindStringSubmatch(part)
		if matches != nil {
			v, _ := strconv.Atoi(matches[1]) // \d+ is always integer
			out = append(out, SliceStep(v))
		} else {
			out = append(out, MapStep(part))
		}
	}

	return out
}

func (v Path) String() string {
	parts := make([]string, 0)
	for _, step := range v {
		var stepStr string
		switch v := step.(type) {
		case MapStep:
			stepStr = v.Key()
		case SliceStep:
			stepStr = fmt.Sprintf("[%d]", v.Index())
		default:
			stepStr = step.String()
		}
		parts = append(parts, stepStr)
	}
	return strings.ReplaceAll(strings.Join(parts, "."), `.[`, `[`)
}

// WithoutFirst returns path without first step or nil.
func (v Path) WithoutFirst() Path {
	if len(v) == 0 {
		return nil
	}
	return v[1:]
}

// WithoutLast returns path without last step or nil.
func (v Path) WithoutLast() Path {
	l := len(v)
	if l == 0 {
		return nil
	}
	return v[0 : l-1]
}

// First returns path first step or nil.
func (v Path) First() Step {
	if len(v) == 0 {
		return nil
	}
	return v[0]
}

// Last returns path last step or nil.
func (v Path) Last() Step {
	l := len(v)
	if l == 0 {
		return nil
	}
	return v[l-1]
}

// Key returns key name.
func (v MapStep) Key() string {
	return string(v)
}

func (v MapStep) String() string {
	return fmt.Sprintf("[%s]", string(v))
}

// Key returns key name.
func (v MapKeyStep) Key() string {
	return string(v)
}

func (v MapKeyStep) String() string {
	return fmt.Sprintf("[%s].<key>", string(v))
}

// Index returns slice index.
func (v SliceStep) Index() int {
	return int(v)
}

func (v SliceStep) String() string {
	return fmt.Sprintf("[%d]", int(v))
}
