package orderedmap

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func (o *OrderedMap) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, key := range o.Keys() {
		// Encode key
		keyNode := &yaml.Node{Kind: yaml.MappingNode}
		if err := keyNode.Encode(key); err != nil {
			return nil, err
		}

		// Encode value
		value, _ := o.Get(key)
		valueNode, err := encodeYamlValue(value)
		if err != nil {
			return nil, err
		}

		// Move head comment from the value to the key node, if any
		if valueNode.HeadComment != "" {
			keyNode.HeadComment = valueNode.HeadComment
			valueNode.HeadComment = ""
		}

		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}

func (o *OrderedMap) UnmarshalYAML(node *yaml.Node) error {
	if o.values == nil {
		o.values = map[string]any{}
	}

	// Check node type
	if node.Tag != "!!map" {
		return fmt.Errorf("cannot unmarshal %s `%s` into orderedmap", node.Tag, node.Value)
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

func encodeYamlValue(value any) (out *yaml.Node, err error) {
	switch v := value.(type) {
	case *yaml.Node:
		return v, nil
	case *OrderedMap:
		if subNode, err := v.MarshalYAML(); err == nil {
			return subNode.(*yaml.Node), nil
		} else {
			return nil, err
		}
	case []any:
		out = &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range v {
			if subNode, err := encodeYamlValue(item); err == nil {
				out.Content = append(out.Content, subNode)
			} else {
				return nil, err
			}
		}
		return out, nil
	default:
		out = &yaml.Node{Kind: yaml.ScalarNode}
		if err := out.Encode(value); err != nil {
			return nil, err
		}
		return out, nil
	}
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
