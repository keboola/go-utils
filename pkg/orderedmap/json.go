package orderedmap

import (
	"bytes"
	"encoding/json"
)

// MarshalJSON implements JSON encoding.
func (o OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	encoder := json.NewEncoder(&buf)
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// add key
		if err := encoder.Encode(k); err != nil {
			return nil, err
		}
		buf.WriteByte(':')
		// add value
		if err := encoder.Encode(o.values[k]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON implements JSON decoding.
func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	if o.values == nil {
		o.values = map[string]any{}
	}
	err := json.Unmarshal(b, &o.values)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	o.keys = make([]string, 0, len(o.values))
	return decodeJsonOrderedMap(dec, o)
}

func decodeJsonOrderedMap(dec *json.Decoder, o *OrderedMap) error {
	hasKey := make(map[string]bool, len(o.values))
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return nil
		}
		key := token.(string)
		if hasKey[key] {
			// duplicate key
			for j, k := range o.keys {
				if k == key {
					copy(o.keys[j:], o.keys[j+1:])
					break
				}
			}
			o.keys[len(o.keys)-1] = key
		} else {
			hasKey[key] = true
			o.keys = append(o.keys, key)
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := o.values[key].(map[string]any); ok {
					newMap := &OrderedMap{
						keys:   make([]string, 0, len(values)),
						values: values,
					}
					if err = decodeJsonOrderedMap(dec, newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if oldMap, ok := o.values[key].(*OrderedMap); ok {
					newMap := &OrderedMap{
						keys:   make([]string, 0, len(oldMap.values)),
						values: oldMap.values,
					}
					if err = decodeJsonOrderedMap(dec, newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if err = decodeJsonOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if values, ok := o.values[key].([]any); ok {
					if err = decodeJsonSlice(dec, values); err != nil {
						return err
					}
					// convert []any of strings to []string
					o.values[key] = convertStringSliceIfPossible(o.values[key])
				} else if err = decodeJsonSlice(dec, []any{}); err != nil {
					return err
				} else {
					// ensure conversion even if initial type hint was empty []any
					if v, ok := o.values[key].([]any); ok {
						o.values[key] = convertStringSliceIfPossible(v)
					}
				}
			}
		}
	}
}

func decodeJsonSlice(dec *json.Decoder, s []any) error {
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if index < len(s) {
					if values, ok := s[index].(map[string]any); ok {
						newMap := &OrderedMap{
							keys:   make([]string, 0, len(values)),
							values: values,
						}
						if err = decodeJsonOrderedMap(dec, newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(*OrderedMap); ok {
						newMap := &OrderedMap{
							keys:   make([]string, 0, len(oldMap.values)),
							values: oldMap.values,
						}
						if err = decodeJsonOrderedMap(dec, newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decodeJsonOrderedMap(dec, &OrderedMap{}); err != nil {
						return err
					}
				} else if err = decodeJsonOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]any); ok {
						if err = decodeJsonSlice(dec, values); err != nil {
							return err
						}
						// convert nested []any of strings to []string
						s[index] = convertStringSliceIfPossible(values)
					} else if err = decodeJsonSlice(dec, []any{}); err != nil {
						return err
					}
				} else if err = decodeJsonSlice(dec, []any{}); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}

// convertStringSliceIfPossible turns []any where all elements are strings into []string recursively.
func convertStringSliceIfPossible(v any) any {
	arr, ok := v.([]any)
	if !ok {
		return v
	}
	outStr := make([]string, len(arr))
	for i, el := range arr {
		switch t := el.(type) {
		case string:
			outStr[i] = t
		default:
			return convertNestedSlices(arr)
		}
	}
	return outStr
}

func convertNestedSlices(arr []any) []any {
	out := make([]any, len(arr))
	for i, el := range arr {
		// Recurse for nested []any
		if sub, ok := el.([]any); ok {
			out[i] = convertStringSliceIfPossible(sub)
			continue
		}
		out[i] = el
	}
	return out
}
