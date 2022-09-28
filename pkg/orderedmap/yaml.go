package orderedmap

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func (o *OrderedMap) UnmarshalYAML(node *yaml.Node) error {
	if o.values == nil {
		o.values = map[string]any{}
	}

	// Iterate nodes: key1, value1, key2, value2, ...
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Tag != "!!str" {
			return fmt.Errorf("expected a string key but got %s on line %d", keyNode.Tag, keyNode.Line)
		}

		// Decode key
		var key string
		if err := keyNode.Decode(&key); err != nil {
			return err
		}

		// Decode value
		var value any
		if err := decodeYamlValue(valueNode, &value); err != nil {
			return err
		}

		// Set to map
		o.Delete(key) // to keep order of duplicate keys
		o.Set(key, value)
	}

	return nil
}

func decodeYamlValue(node *yaml.Node, out *any) error {
	switch node.Tag {
	case "!!map": // key-value map
		outMap := New()
		if err := node.Decode(outMap); err != nil {
			return err
		}
		*out = outMap
	case "!!seq": // array
		outSlice := make([]any, 0)
		for _, item := range node.Content {
			var itemValue any
			if err := decodeYamlValue(item, &itemValue); err != nil {
				return err
			}
			outSlice = append(outSlice, itemValue)
		}
		*out = outSlice
	default:
		if err := node.Decode(out); err != nil {
			return err
		}
	}
	return nil
}
