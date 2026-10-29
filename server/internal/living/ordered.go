package living

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// OrderedObject is a JSON/YAML object that preserves key order (unlock / authoring order).
type OrderedObject struct {
	Keys   []string
	Values map[string]any
}

func newOrderedObject() OrderedObject {
	return OrderedObject{Keys: nil, Values: map[string]any{}}
}

func (o *OrderedObject) Set(key string, value any) {
	if o.Values == nil {
		o.Values = map[string]any{}
	}
	if _, exists := o.Values[key]; !exists {
		o.Keys = append(o.Keys, key)
	}
	o.Values[key] = value
}

func (o OrderedObject) Get(key string) (any, bool) {
	if o.Values == nil {
		return nil, false
	}
	v, ok := o.Values[key]
	return v, ok
}

func (o OrderedObject) Len() int {
	return len(o.Keys)
}

func (o OrderedObject) MarshalJSON() ([]byte, error) {
	if o.Values == nil {
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.Keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := json.Marshal(o.Values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (o *OrderedObject) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = OrderedObject{}
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("ordered object: expected '{', got %v", tok)
	}
	out := newOrderedObject()
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("ordered object: expected string key, got %T", keyTok)
		}
		var value any
		if err := dec.Decode(&value); err != nil {
			return err
		}
		out.Set(key, value)
	}
	tok, err = dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("ordered object: expected '}', got %v", tok)
	}
	*o = out
	return nil
}
