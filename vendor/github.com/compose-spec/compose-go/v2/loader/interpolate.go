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

package loader

import (
	"fmt"
	"strconv"
	"strings"

	interp "github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/sirupsen/logrus"
)

var interpolateTypeCastMapping = buildInterpolateTypeCastMapping()

// buildInterpolateTypeCastMapping registers numeric/boolean casts per layer of
// the specification: container_spec attributes apply wherever a container is
// declared (services, jobs, pre_start init containers), workload_spec
// attributes to services and jobs, service-only attributes to services.
func buildInterpolateTypeCastMapping() map[tree.Path]interp.Cast {
	casts := map[tree.Path]interp.Cast{
		iPath("networks", tree.PathMatchAll, "external"):    toBoolean,
		iPath("networks", tree.PathMatchAll, "internal"):    toBoolean,
		iPath("networks", tree.PathMatchAll, "attachable"):  toBoolean,
		iPath("networks", tree.PathMatchAll, "enable_ipv4"): toBoolean,
		iPath("networks", tree.PathMatchAll, "enable_ipv6"): toBoolean,
		iPath("volumes", tree.PathMatchAll, "external"):     toBoolean,
		iPath("secrets", tree.PathMatchAll, "external"):     toBoolean,
		iPath("configs", tree.PathMatchAll, "external"):     toBoolean,
	}
	containerSpec := []tree.Path{
		iPath("services", tree.PathMatchAll),
		iPath("jobs", tree.PathMatchAll),
		iPath("services", tree.PathMatchAll, "pre_start", tree.PathMatchAll),
	}
	workloadSpec := []tree.Path{
		iPath("services", tree.PathMatchAll),
		iPath("jobs", tree.PathMatchAll),
	}
	serviceOnly := []tree.Path{
		iPath("services", tree.PathMatchAll),
	}
	add := func(prefixes []tree.Path, cast interp.Cast, parts ...string) {
		for _, prefix := range prefixes {
			p := prefix
			for _, part := range parts {
				p = p.Next(part)
			}
			casts[p] = cast
		}
	}

	add(containerSpec, toInt64, "cpu_count")
	add(containerSpec, toFloat, "cpu_percent")
	add(containerSpec, toInt64, "cpu_period")
	add(containerSpec, toInt64, "cpu_quota")
	add(containerSpec, toInt64, "cpu_rt_period")
	add(containerSpec, toInt64, "cpu_rt_runtime")
	add(containerSpec, toFloat32, "cpus")
	add(containerSpec, toInt64, "cpu_shares")
	add(containerSpec, toBoolean, "init")
	add(containerSpec, toBoolean, "oom_kill_disable")
	add(containerSpec, toInt64, "oom_score_adj")
	add(containerSpec, toInt64, "pids_limit")
	add(containerSpec, toBoolean, "privileged")
	add(containerSpec, toBoolean, "read_only")
	add(containerSpec, toInt, "ulimits", tree.PathMatchAll)
	add(containerSpec, toInt, "ulimits", tree.PathMatchAll, "hard")
	add(containerSpec, toInt, "ulimits", tree.PathMatchAll, "soft")
	add(containerSpec, toBoolean, "volumes", tree.PathMatchList, "read_only")
	add(containerSpec, toBoolean, "volumes", tree.PathMatchList, "volume", "nocopy")

	add(workloadSpec, toBoolean, "depends_on", tree.PathMatchAll, "required")
	add(workloadSpec, toBoolean, "depends_on", tree.PathMatchAll, "restart")
	add(workloadSpec, toInt, "healthcheck", "retries")
	add(workloadSpec, toBoolean, "healthcheck", "disable")
	add(workloadSpec, toInt, "ports", tree.PathMatchList, "target")
	add(workloadSpec, toBoolean, "stdin_open")
	add(workloadSpec, toBoolean, "tty")

	add(serviceOnly, toInt, "deploy", "replicas")
	add(serviceOnly, toInt, "deploy", "update_config", "parallelism")
	add(serviceOnly, toFloat, "deploy", "update_config", "max_failure_ratio")
	add(serviceOnly, toInt, "deploy", "rollback_config", "parallelism")
	add(serviceOnly, toFloat, "deploy", "rollback_config", "max_failure_ratio")
	add(serviceOnly, toInt, "deploy", "restart_policy", "max_attempts")
	add(serviceOnly, toInt, "deploy", "placement", "max_replicas_per_node")
	add(serviceOnly, toInt, "scale")

	return casts
}

func iPath(parts ...string) tree.Path {
	return tree.NewPath(parts...)
}

func toInt(value string) (interface{}, error) {
	return strconv.Atoi(value)
}

func toInt64(value string) (interface{}, error) {
	return strconv.ParseInt(value, 10, 64)
}

func toFloat(value string) (interface{}, error) {
	return strconv.ParseFloat(value, 64)
}

func toFloat32(value string) (interface{}, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return nil, err
	}
	return float32(f), nil
}

// should match http://yaml.org/type/bool.html
func toBoolean(value string) (interface{}, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "y", "yes", "on":
		logrus.Warnf("%q for boolean is not supported by YAML 1.2, please use `true`", value)
		return true, nil
	case "n", "no", "off":
		logrus.Warnf("%q for boolean is not supported by YAML 1.2, please use `false`", value)
		return false, nil
	default:
		return nil, fmt.Errorf("invalid boolean: %s", value)
	}
}
