package registry

import (
	"context"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/graph"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

type LockType int

const (
	InUse                LockType = 0
	InUseByOtherInstance LockType = 1
	InUseCyclic          LockType = 2
	Locked               LockType = 3
)

var (
	configMapName = "devspace-dependencies"
)

type ownership struct {
	Server string `yaml:"server,omitempty" json:"server,omitempty"`
	RunID  string `yaml:"runID,omitempty" json:"runID,omitempty"`
}

type DependencyRegistry interface {
	// TryLockDependencies tries to lock the given dependencies and returns the dependencies that were locked
	TryLockDependencies(ctx devspacecontext.Context, fromDependency string, dependencyNames []string, forceLeader bool) (map[string]LockType, error)

	// SetServer sets the target server
	SetServer(server string)
}

func NewDependencyRegistry(name string, mock bool) DependencyRegistry {
	return &dependencyRegistry{
		mock:               mock,
		interProcess:       NewInterProcessCommunicator(),
		dependencyGraph:    graph.NewGraph(graph.NewNode(name, nil)),
		lockedDependencies: map[string]LockType{},
	}
}

type dependencyRegistry struct {
	server       string
	mock         bool
	interProcess InterProcess

	// dependencyGraph is used to detect cyclic dependencies
	dependencyGraph *graph.Graph

	lockedDependenciesLock sync.Mutex
	lockedDependencies     map[string]LockType
}

func (d *dependencyRegistry) SetServer(server string) {
	d.lockedDependenciesLock.Lock()
	defer d.lockedDependenciesLock.Unlock()

	d.server = server
}

func (d *dependencyRegistry) TryLockDependencies(ctx devspacecontext.Context, fromDependency string, dependencyNames []string, forceLeader bool) (lockedDependencies map[string]LockType, err error) {
	d.lockedDependenciesLock.Lock()
	defer d.lockedDependenciesLock.Unlock()

	// was already excluded
	lockedDependencies = map[string]LockType{}
	for _, dependencyName := range dependencyNames {
		// special case if we want to initialize the lock tree
		if d.dependencyGraph.Root.ID == fromDependency && d.dependencyGraph.Root.ID == dependencyName && forceLeader {
			lockType, ok := d.lockedDependencies[dependencyName]
			if !ok {
				lockedDependencies[dependencyName] = Locked
			} else {
				lockedDependencies[dependencyName] = lockType
			}

			continue
		}

		// would locking this dependency create a circle?
		_, err := d.dependencyGraph.InsertNodeAt(fromDependency, dependencyName, nil)
		if err != nil {
			if _, ok := err.(*graph.CyclicError); !ok {
				return nil, err
			}

			lockedDependencies[dependencyName] = InUseCyclic
		} else {
			lockType, ok := d.lockedDependencies[dependencyName]
			if !ok {
				lockedDependencies[dependencyName] = Locked
			} else {
				lockedDependencies[dependencyName] = lockType
			}
		}
	}

	// exclude the dependencies
	if !d.mock {
		err = d.lockDependencies(ctx, lockedDependencies, forceLeader, 4)
		if err != nil {
			return nil, err
		}
	}

	// update lock map
	for dependencyName, lockType := range lockedDependencies {
		if lockType == Locked || lockType == InUseCyclic {
			lockType = InUse
		}

		d.lockedDependencies[dependencyName] = lockType
	}

	return lockedDependencies, nil
}

func (d *dependencyRegistry) lockDependencies(ctx devspacecontext.Context, lockedDependencies map[string]LockType, forceLeader bool, retries int) error {
	if ctx.KubeClient() == nil {
		return nil
	}

	// check if there is at least 1 locked dependency
	found := false
	for _, lockType := range lockedDependencies {
		if lockType == Locked {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	// encode server and run id
	encoded, _ := yaml.Marshal(&ownership{
		Server: d.server,
		RunID:  ctx.RunID(),
	})

	// check configmap if the dependency is excluded
	configMap, err := ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Get(ctx.Context(), configMapName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: ctx.KubeClient().Namespace(),
			},
			Data: map[string]string{},
		}
		for dependencyName, lockType := range lockedDependencies {
			if lockType != Locked {
				continue
			}

			configMap.Data[dependencyName] = string(encoded)
		}

		_, err = ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Create(ctx.Context(), configMap, metav1.CreateOptions{})
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				if retries == 0 {
					return err
				}

				return d.lockDependencies(ctx, lockedDependencies, forceLeader, retries-1)
			}

			return err
		}

		return nil
	}

	// check which dependencies are taken by us vs. which we should take over
	shouldUpdate := false
	failedPings := map[string]bool{}
	for dependencyName, lockType := range lockedDependencies {
		if lockType != Locked {
			continue
		}

		if configMap.Data == nil || configMap.Data[dependencyName] == "" {
			configMap.Data[dependencyName] = string(encoded)
			shouldUpdate = true
			continue
		}

		// decode the payload
		payload := &ownership{}
		err = yaml.Unmarshal([]byte(configMap.Data[dependencyName]), payload)
		if err != nil {
			ctx.Log().Debugf("error decoding ownership from configmap: %v", err)
			configMap.Data[dependencyName] = string(encoded)
			shouldUpdate = true
			continue
		} else if payload.Server == "" || payload.RunID == "" {
			ctx.Log().Debugf("server or run id missing in configmap payload")
			configMap.Data[dependencyName] = string(encoded)
			shouldUpdate = true
			continue
		}

		// check if we self have ownership of the dependency
		if payload.RunID == ctx.RunID() {
			lockedDependencies[dependencyName] = InUse
			continue
		}

		// somebody else has ownership
		// check ping cache
		if failedPings[payload.Server] {
			configMap.Data[dependencyName] = string(encoded)
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
				shouldUpdate = true
				continue
			}
		}

		// already in use
		lockedDependencies[dependencyName] = InUseByOtherInstance
	}

	// check if we should update the configmap
	if shouldUpdate {
		_, err = ctx.KubeClient().KubeClient().CoreV1().ConfigMaps(ctx.KubeClient().Namespace()).Update(ctx.Context(), configMap, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				if retries == 0 {
					return err
				}

				return d.lockDependencies(ctx, lockedDependencies, forceLeader, retries-1)
			}

			return err
		}
	}

	return nil
}
