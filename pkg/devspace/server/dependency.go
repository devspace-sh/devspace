package server

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"net/http"
)

func (h *handler) ping(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t registry.PingPayload
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if t.RunID != h.ctx.RunID() {
		http.Error(w, h.ctx.RunID(), http.StatusConflict)
		return
	}
}

func (h *handler) excludeDependency(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t registry.ExcludePayload
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.pipeline == nil || t.RunID != h.ctx.RunID() {
		// we allow this here as apparently the request targeted a wrong server
		return
	}

	// we don't allow killing ourselves
	if h.pipeline.Name() == t.DependencyName {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// try to find the dependency name and kill it
	dep := findDependency(h.pipeline, t.DependencyName)
	if dep != nil {
		h.ctx.Log().Debugf("stopping dependency %v", t.DependencyName)
		err = dep.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("error stopping dependency: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func findDependency(pipe types.Pipeline, dependencyName string) types.Pipeline {
	dependencies := pipe.Dependencies()
	for _, dep := range dependencies {
		if dep.Name() == dependencyName {
			return dep
		}

		pipeline := findDependency(dep, dependencyName)
		if pipeline != nil {
			return pipeline
		}
	}

	return nil
}
