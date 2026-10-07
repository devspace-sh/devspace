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

// JobConfig is the configuration of one job
type JobConfig struct {
	Name     string         `yaml:"name,omitempty" json:"-"`
	Profiles []string       `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Triggers *TriggerConfig `yaml:"triggers,omitempty" json:"triggers,omitempty"`

	ContainerSpec `yaml:",inline" mapstructure:",squash"`
	WorkloadSpec  `yaml:",inline" mapstructure:",squash"`

	Extensions Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// TriggerConfig defines trigger conditions for a job.
// Manual is tri-state: nil (unset) leaves manual execution allowed — any job
// can be triggered by an explicit run command; an explicit false forbids it;
// an explicit true declares the job as manual-only intent.
type TriggerConfig struct {
	Manual     *bool            `yaml:"manual,omitempty" json:"manual,omitempty"`
	Schedule   []ScheduleConfig `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Extensions Extensions       `yaml:"#extensions,inline,omitempty" json:"-"`
}

// ScheduleConfig defines a schedule for a job trigger.
// A plain crontab expression in yaml is canonicalized into a ScheduleConfig
// with only Cron set.
type ScheduleConfig struct {
	Cron        string     `yaml:"cron,omitempty" json:"cron,omitempty"`
	Timezone    string     `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	Concurrency string     `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	MissedFires string     `yaml:"missed_fires,omitempty" json:"missed_fires,omitempty"`
	Extensions  Extensions `yaml:"#extensions,inline,omitempty" json:"-"`
}

// Jobs is a mapping of job names to job configurations
type Jobs map[string]JobConfig
