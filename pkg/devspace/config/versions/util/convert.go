package util

import "encoding/json"

// Convert converts the old object into the new object through json serialization / deserialization
func Convert(old interface{}, new interface{}) error {
	o, err := json.Marshal(old)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(o, &new); err != nil {
		return err
	}
	return nil
}
