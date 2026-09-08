/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package types

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/docker/go-units"
	"go.yaml.in/yaml/v4"
)

// UnitBytes is the bytes type
type UnitBytes int64

// MarshalYAML makes UnitBytes implement yaml.Marshaller
func (u UnitBytes) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%d", u), nil
}

// MarshalJSON makes UnitBytes implement json.Marshaler
func (u UnitBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%d"`, u)), nil
}

// parseString parses a string into a UnitBytes value, supporting plain
// integers, negative values (e.g., "-1"), and human-readable byte units
// (e.g., "1g", "512m").
func (u *UnitBytes) parseString(s string) error {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*u = UnitBytes(n)
		return nil
	}
	b, err := units.RAMInBytes(s)
	*u = UnitBytes(b)
	return err
}

// UnmarshalJSON makes UnitBytes implement json.Unmarshaler
func (u *UnitBytes) UnmarshalJSON(data []byte) error {
	var v int64
	if err := json.Unmarshal(data, &v); err == nil {
		*u = UnitBytes(v)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return u.parseString(s)
}

// UnmarshalYAML makes UnitBytes implement yaml.Unmarshaler
func (u *UnitBytes) UnmarshalYAML(value *yaml.Node) error {
	var v int64
	if err := value.Decode(&v); err == nil {
		*u = UnitBytes(v)
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	return u.parseString(s)
}

func (u *UnitBytes) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case int:
		*u = UnitBytes(v)
	case string:
		return u.parseString(v)
	}
	return nil
}
