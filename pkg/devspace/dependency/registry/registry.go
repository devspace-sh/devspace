package registry

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

var (
	configMapName = "devspace-dependencies"
)

type ownership struct {
	Server string `yaml:"server,omitempty" json:"server,omitempty"`
	RunID  string `yaml:"runID,omitempty" json:"runID,omitempty"`
}

type DependencyRegistry interface {
	// ForceExclude will immediately exclude the dependency
	ForceExclude(dependencyName string)

	// MarkDependencyExcluded excludes the dependency if it wasn't already for the run
	// and returns if the dependency was excluded before
	MarkDependencyExcluded(ctx devspacecontext.Context, dependencyName string, forceLeader bool) (bool, error)

	// MarkDependenciesExcluded same as MarkDependencyExcluded but for multiple dependencies
	MarkDependenciesExcluded(ctx devspacecontext.Context, dependencyNames []string, forceLeader bool) (map[string]bool, error)

	// OwnedDependency signals if we are the owner of the dependency
	OwnedDependency(dependencyName string) bool

	// SetServer sets the target server
	SetServer(server string)
}

func NewDependencyRegistry(mock bool) DependencyRegistry {
	return &dependencyRegistry{
		mock:                 mock,
		interProcess:         NewInterProcessCommunicator(),
		excludedDependencies: map[string]bool{},
		ownedDependencies:    map[string]bool{},
	}
}

type dependencyRegistry struct {
	server       string
	mock         bool
	interProcess InterProcess

	excludedDependenciesLock sync.Mutex
	excludedDependencies     map[string]bool
	ownedDependencies        map[string]bool
}

func (d *dependencyRegistry) OwnedDependency(dependencyName string) bool {
	d.excludedDependenciesLock.Lock()
	defer d.excludedDependenciesLock.Unlock()

	return d.ownedDependencies[dependencyName]
}

func (d *dependencyRegistry) SetServer(server string) {
	d.excludedDependenciesLock.Lock()
	defer d.excludedDependenciesLock.Unlock()

	d.server = server
}

func (d *dependencyRegistry) ForceExclude(dependencyName string) {
	d.excludedDependenciesLock.Lock()
	defer d.excludedDependenciesLock.Unlock()

	d.excludedDependencies[dependencyName] = true
}

func (d *dependencyRegistry) MarkDependencyExcluded(ctx devspacecontext.Context, dependencyName string, forceLeader bool) (bool, error) {
	excluded, err := d.MarkDependenciesExcluded(ctx, []string{dependencyName}, forceLeader)
	if err != nil {
		return false, err
	}

	return excluded[dependencyName], nil
}

func (d *dependencyRegistry) MarkDependenciesExcluded(ctx devspacecontext.Context, dependencyNames []string, forceLeader bool) (map[string]bool, error) {
	d.excludedDependenciesLock.Lock()
	defer d.excludedDependenciesLock.Unlock()

	// was already excluded
	filteredDependencyNames := []string{}
	for _, dependencyName := range dependencyNames {
		if !d.excludedDependencies[dependencyName] {
			filteredDependencyNames = append(filteredDependencyNames, dependencyName)
		}
	}

	// all dependencies were excluded already
	if len(filteredDependencyNames) == 0 {
		return map[string]bool{}, nil
	}

	// exclude the dependencies
	retMap := map[string]bool{}
	if !d.mock {
		var err error
		retMap, err = d.excludeDependencies(ctx, filteredDependencyNames, forceLeader, 4)
		if err != nil {
			return nil, err
		}
	}

	// now exclude all dependencies
	for _, dependencyName := range filteredDependencyNames {
		if d.mock {
			retMap[dependencyName] = true
		}

		d.excludedDependencies[dependencyName] = true
	}

	// now mark the dependencies we have excluded
	for dependencyName := range retMap {
		d.ownedDependencies[dependencyName] = true
	}

	return retMap, nil
}

func (d *dependencyRegistry) excludeDependencies(ctx devspacecontext.Context, dependencyNames []string, forceLeader bool, retries int) (map[string]bool, error) {
	retMap := map[string]bool{}
	if len(dependencyNames) == 0 || ctx.KubeClient() == nil {
		return retMap, nil
	}

	encoded, _ := yaml.Marshal(&ownership{
		Server: d.server,
		RunID:  ctx.RunID(),
	})

	// check configmap if the dependency is excluded
	configMap, err := ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Get(ctx.Context(), configMapName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: ctx.KubeClient().Namespace(),
			},
			Data: map[string]string{},
		}
		for _, dependencyName := range dependencyNames {
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
		}

		_, err = ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Create(ctx.Context(), configMap, metav1.CreateOptions{})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				if retries == 0 {
					return nil, err
				}

				return d.excludeDependencies(ctx, dependencyNames, forceLeader, retries-1)
			}

			return nil, err
		}

		return retMap, nil
	}

	// check which dependencies are taken by us vs. which we should take over
	shouldUpdate := false
	failedPings := map[string]bool{}
	for _, dependencyName := range dependencyNames {
		if configMap.Data == nil || configMap.Data[dependencyName] == "" {
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
			shouldUpdate = true
			continue
		}

		// decode the payload
		payload := &ownership{}
		err = yaml.Unmarshal([]byte(configMap.Data[dependencyName]), payload)
		if err != nil {
			ctx.Log().Debugf("error decoding ownership from configmap: %v", err)
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
			shouldUpdate = true
			continue
		} else if payload.Server == "" || payload.RunID == "" {
			ctx.Log().Debugf("server or run id missing in configmap payload")
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
			shouldUpdate = true
			continue
		}

		// check if we self have ownership of the dependency
		if payload.RunID == ctx.RunID() {
			continue
		}

		// somebody else has ownership
		// check ping cache
		if failedPings[payload.Server] {
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
			shouldUpdate = true
			continue
		}

		// try pinging the other instance
		pingCtx, pingCancel := context.WithTimeout(ctx.Context(), time.Second*2)
		pinged, err := d.interProcess.Ping(pingCtx, payload.Server, &PingPayload{
			RunID: payload.RunID,
		})
		pingCancel()
		if !pinged || err != nil {
			if err != nil {
				ctx.Log().Debugf("error pinging server: %v", err)
			}
			failedPings[payload.Server] = true
			configMap.Data[dependencyName] = string(encoded)
			retMap[dependencyName] = true
			shouldUpdate = true
			continue
		}

		// check if we should take over
		if forceLeader {
			excludeCtx, excludeCancel := context.WithTimeout(ctx.Context(), time.Second*10)
			allowed, err := d.interProcess.ExcludeDependency(excludeCtx, payload.Server, &ExcludePayload{
				RunID:          payload.RunID,
				DependencyName: dependencyName,
			})
			excludeCancel()
			if err != nil || allowed {
				if err != nil {
					ctx.Log().Debugf("error taking over dependency: %v", err)
				}

				configMap.Data[dependencyName] = string(encoded)
				retMap[dependencyName] = true
				shouldUpdate = true
				continue
			}
		}
	}

	// check if we should update the configmap
	if shouldUpdate {
		_, err = ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Update(ctx.Context(), configMap, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				if retries == 0 {
					return nil, err
				}

				return d.excludeDependencies(ctx, dependencyNames, forceLeader, retries-1)
			}

			return nil, err
		}
	}

	return retMap, nil
}
