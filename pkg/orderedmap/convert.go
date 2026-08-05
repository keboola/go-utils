package orderedmap

import (
	"fmt"
	"slices"
)

// ToStringSlice converts a value obtained from OrderedMap (e.g. via Get or GetNested) into a
// []string. OrderedMap itself never guesses a field's schema from its runtime content - JSON/YAML
// arrays always decode to []any - so callers that know a specific field is a string array must
// opt in explicitly via this helper. Accepts either []any (the JSON/YAML-decoded shape) or
// []string directly (e.g. a value set programmatically via Set).
//
// A nil value is an error, not an empty slice: nil is what GetNestedOrNil returns for a missing
// path, and treating it as "no error, empty result" would recreate the exact silent-no-op this
// helper exists to prevent. Callers must check Get/GetNested's found return first.
func ToStringSlice(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		// Clone: the []any branch below always returns a fresh slice, so callers must be able to
		// mutate the result without risk regardless of which branch produced it. Without this,
		// mutating the result of a []string-backed value would alias and mutate the OrderedMap.
		return slices.Clone(v), nil
	case []any:
		out := make([]string, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf(`expected string at index %d, found "%T"`, i, item)
			}
			out[i] = s
		}
		return out, nil
	default:
		return nil, fmt.Errorf(`expected []any or []string, found "%T"`, value)
	}
}
