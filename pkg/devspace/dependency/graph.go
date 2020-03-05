package dependency

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type graph struct {
	Nodes map[string]*node

	Root *node
}

func newGraph(root *node) *graph {
	graph := &graph{
		Nodes: make(map[string]*node),
		Root:  root,
	}

	graph.Nodes[root.ID] = root
	return graph
}

// node is a node in a graph
type node struct {
	ID   string
	Data interface{}

	parents []*node
	childs  []*node
}

func newNode(id string, data interface{}) *node {
	return &node{
		ID:   id,
		Data: data,

		parents: make([]*node, 0, 1),
		childs:  make([]*node, 0, 1),
	}
}

// insertNodeAt inserts a new node at the given parent position
func (g *graph) insertNodeAt(parentID string, id string, data interface{}) (*node, error) {
	parentNode, ok := g.Nodes[parentID]
	if !ok {
		return nil, errors.Errorf("Parent %s does not exist", parentID)
	}
	if existingNode, ok := g.Nodes[id]; ok {
		err := g.addEdge(parentNode.ID, existingNode.ID)
		if err != nil {
			return nil, err
		}

		return existingNode, nil
	}

	node := newNode(id, data)

	g.Nodes[node.ID] = node

	parentNode.childs = append(parentNode.childs, node)
	node.parents = append(node.parents, parentNode)

	return node, nil
}

// removeNode removes a node with no children in the graph
func (g *graph) removeNode(id string) error {
	if node, ok := g.Nodes[id]; ok {
		if len(node.childs) > 0 {
			return errors.Errorf("Cannot remove %s from graph because it has still children", id)
		}

		// Remove child from parents
		for _, parent := range node.parents {
			i := -1
			for idx, c := range parent.childs {
				if c.ID == id {
					i = idx
				}
			}

			if i != -1 {
				parent.childs = append(parent.childs[:i], parent.childs[i+1:]...)
			}
		}

		// Remove from graph nodes
		delete(g.Nodes, id)
	}

	return nil
}

// getNextLeaf returns the next leaf in the graph from node start
func (g *graph) getNextLeaf(start *node) *node {
	if len(start.childs) == 0 {
		return start
	}

	return g.getNextLeaf(start.childs[0])
}

// cyclicError is the type that is returned if a cyclic edge would be inserted
type cyclicError struct {
	path []*node
}

// Error implements error interface
func (c *cyclicError) Error() string {
	cycle := []string{c.path[len(c.path)-1].ID}
	for _, node := range c.path {
		cycle = append(cycle, node.ID)
	}

	return fmt.Sprintf("Cyclic dependency found: \n%s", strings.Join(cycle, "\n"))
}

// addEdge adds a new edge from a node to a node and returns an error if it would result in a cyclic graph
func (g *graph) addEdge(fromID string, toID string) error {
	from, ok := g.Nodes[fromID]
	if !ok {
		return errors.Errorf("fromID %s does not exist", fromID)
	}
	to, ok := g.Nodes[toID]
	if !ok {
		return errors.Errorf("toID %s does not exist", toID)
	}

	// Check if cyclic
	path := findFirstPath(to, from)
	if path != nil {
		return &cyclicError{
			path: path,
		}
	}

	// Check if there is already an edge
	for _, child := range from.childs {
		if child.ID == to.ID {
			return nil
		}
	}

	from.childs = append(from.childs, to)
	to.parents = append(to.parents, from)
	return nil
}

// find first path from node to node with DFS
func findFirstPath(from *node, to *node) []*node {
	isVisited := map[string]bool{}
	pathList := []*node{from}

	// Call recursive utility
	if findFirstPathRecursive(from, to, isVisited, &pathList) {
		return pathList
	}

	return nil
}

// A recursive function to print
// all paths from 'u' to 'd'.
// isVisited[] keeps track of
// vertices in current path.
// localPathList<> stores actual
// vertices in the current path
func findFirstPathRecursive(u *node, d *node, isVisited map[string]bool, localPathList *[]*node) bool {
	// Mark the current node
	isVisited[u.ID] = true

	// Is destination?
	if u.ID == d.ID {
		return true
	}

	// Recur for all the vertices
	// adjacent to current vertex
	for _, child := range u.childs {
		if _, ok := isVisited[child.ID]; !ok {
			// store current node
			// in path[]
			*localPathList = append(*localPathList, child)
			if findFirstPathRecursive(child, d, isVisited, localPathList) {
				return true
			}

			// remove current node
			// in path[]
			i := -1
			for idx, c := range *localPathList {
				if c.ID == child.ID {
					i = idx
				}
			}
			if i != -1 {
				*localPathList = append((*localPathList)[:i], (*localPathList)[i+1:]...)
			}
		}
	}

	// Mark the current node
	delete(isVisited, u.ID)
	return false
}
