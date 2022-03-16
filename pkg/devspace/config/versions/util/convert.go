package util

import yaml "gopkg.in/yaml.v3"

// Convert converts the old object into the new object through json serialization / deserialization
func Convert(old interface{}, new interface{}) error {
	o, err := yaml.Marshal(old)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(o, new); err != nil {
		return err
	}
	return nil
}
