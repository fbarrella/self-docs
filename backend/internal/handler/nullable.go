package handler

import (
	"bytes"
	"encoding/json"
)

// nullableString distinguishes an absent JSON field from an explicit null.
// Absent leaves Present false; `"key": null` or `"key": "value"` set Present
// true. This lets PUT requests clear a nullable field such as parent_id.
type nullableString struct {
	Value   *string
	Present bool
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.Present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}
