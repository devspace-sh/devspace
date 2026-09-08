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

// ServiceHook is a command exec'd inside the service container at a lifecycle
// event (post_start, pre_stop).
type ServiceHook struct {
	Command     ShellCommand      `yaml:"command,omitempty" json:"command"`
	User        string            `yaml:"user,omitempty" json:"user,omitempty"`
	Privileged  bool              `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	Environment MappingWithEquals `yaml:"environment,omitempty" json:"environment,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// PreStartHook is an init container run to completion before the service
// starts. It accepts the full container specification, inheriting from the
// service in the spirit of the yaml merge rules: collection attributes
// complete the inherited value with the hook's declarations winning on
// conflicts, scalar attributes replace it (image is inherited via
// normalization when undeclared).
type PreStartHook struct {
	ContainerSpec `yaml:",inline" mapstructure:",squash"`

	// PerReplica runs the hook once per service replica instead of once per
	// service.
	PerReplica bool `yaml:"per_replica,omitempty" json:"per_replica,omitempty"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}
