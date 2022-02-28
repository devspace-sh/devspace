package pipeline

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/graph"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/util/tomb"
)

const (
	rootID = "___root___"
)

type Builder interface {
	BuildPipeline(devPodManager devpod.Manager, configPipeline *latest.Pipeline, registry registry.DependencyRegistry) (Pipeline, error)
}

func NewPipelineBuilder() Builder {
	return &builder{}
}

type builder struct{}

func (b *builder) BuildPipeline(devPodManager devpod.Manager, configPipeline *latest.Pipeline, registry registry.DependencyRegistry) (Pipeline, error) {
	pip := NewPipeline(devPodManager, registry).(*pipeline)
	pip.Jobs = map[string]*PipelineJob{
		"default": {
			Name:          "default",
			DevPodManager: pip.DevPodManager(),
			JobConfig:     &configPipeline.PipelineJob,
			Children:      []*PipelineJob{},
			t:             &tomb.Tomb{},
		},
	}

	// add other jobs
	for k, j := range configPipeline.Jobs {
		pip.Jobs[k] = &PipelineJob{
			Name:          k,
			DevPodManager: pip.DevPodManager(),
			JobConfig:     j,
			Children:      []*PipelineJob{},
			t:             &tomb.Tomb{},
		}
	}

	rootNode := graph.NewNode(rootID, nil)
	jobGraph := graph.NewGraph(rootNode)

	// add the jobs that have no dependencies
	leftJobs := map[string]*PipelineJob{}
	for k, j := range pip.Jobs {
		leftJobs[k] = j
	}
	for {
		changed := false
		for k, j := range leftJobs {
			foundAllAfter := true
			for _, after := range j.JobConfig.After {
				if pip.Jobs[after] == nil {
					return nil, fmt.Errorf("job %s in pipeline has wrong after job %s: this job does not exist", k, after)
				}
				if k == after {
					return nil, fmt.Errorf("error in job %s: cannot use itself as after", k)
				}
				if _, ok := jobGraph.Nodes[after]; !ok {
					foundAllAfter = false
					break
				}
			}
			if foundAllAfter {
				if len(j.JobConfig.After) > 0 {
					_, err := jobGraph.InsertNodeAt(j.JobConfig.After[0], k, j)
					if err != nil {
						return nil, resolveCyclicError(err)
					}

					// add other afters as edges
					for _, after := range j.JobConfig.After[1:] {
						err = jobGraph.AddEdge(after, k)
						if err != nil {
							return nil, resolveCyclicError(err)
						}
					}
				} else {
					_, err := jobGraph.InsertNodeAt(rootNode.ID, k, j)
					if err != nil {
						return nil, resolveCyclicError(err)
					}
				}

				delete(leftJobs, k)
				changed = true
			}
		}

		// are we done?
		if len(leftJobs) == 0 {
			break
		}

		// we are stuck, seems like there is no solution
		// to solve this graph
		if !changed {
			problematicJobs := []string{}
			for k := range leftJobs {
				problematicJobs = append(problematicJobs, k)
			}

			return nil, fmt.Errorf("unable to solve direct graph for pipeline. Seems like you have specified the following jobs that depend on each other: %s", problematicJobs)
		}
	}

	// resolve the pipeline
	return pip, addRecursive(rootNode, &pip.JobsPipeline)
}

func addRecursive(node *graph.Node, childs *[]*PipelineJob) error {
	for _, c := range node.Childs {
		job := c.Data.(*PipelineJob)
		*childs = append(*childs, job)

		if job.JobConfig.Rerun != nil && len(c.Childs) > 0 {
			return fmt.Errorf("rerun is not supported for jobs where other jobs depend on. Please only use rerun if there is no other job referencing it with 'after'. Rerun job where others depend on: %s", c.ID)
		}

		job.Parents = []*PipelineJob{}
		for _, parent := range c.Parents {
			if parent.ID == rootID {
				continue
			}

			p := parent.Data.(*PipelineJob)
			job.Parents = append(job.Parents, p)
		}

		err := addRecursive(c, &job.Children)
		if err != nil {
			return err
		}
	}

	return nil
}

func resolveCyclicError(err error) error {
	if cyclicErr, ok := err.(*graph.CyclicError); ok {
		cyclicErr.What = "Job"
		return cyclicErr
	}

	return fmt.Errorf("error constructing pipeline: %v", err)
}
