package dependency

import (
	"testing"
)

func TestGraph(t *testing.T) {
	var (
		root                   = newNode("root", nil)
		rootChild1             = newNode("rootChild1", nil)
		rootChild2             = newNode("rootChild2", nil)
		rootChild3             = newNode("rootChild3", nil)
		rootChild2Child1       = newNode("rootChild2Child1", nil)
		rootChild2Child1Child1 = newNode("rootChild2Child1Child1", nil)

		testGraph = newGraph(root)
	)

	_, err := testGraph.insertNodeAt("does not exits", rootChild1.ID, nil)
	if err == nil {
		t.Fatal("insertNodeAt error expected")
	}

	testGraph.insertNodeAt(root.ID, rootChild1.ID, nil)
	testGraph.insertNodeAt(root.ID, rootChild2.ID, nil)
	testGraph.insertNodeAt(root.ID, rootChild3.ID, nil)

	testGraph.insertNodeAt(rootChild2.ID, rootChild2Child1.ID, nil)
	testGraph.insertNodeAt(rootChild2Child1.ID, rootChild2Child1Child1.ID, nil)
	testGraph.insertNodeAt(rootChild3.ID, rootChild2.ID, nil)

	// Cyclic graph error
	_, err = testGraph.insertNodeAt(rootChild2Child1Child1.ID, rootChild3.ID, nil)
	if err == nil {
		t.Fatal("Cyclic error expected")
	} else {
		errMsg := `Cyclic dependency found: 
rootChild2Child1Child1
rootChild3
rootChild2
rootChild2Child1
rootChild2Child1Child1`

		if err.Error() != errMsg {
			t.Fatalf("Expected %s, got %s", errMsg, err.Error())
		}
	}

	// Find first path
	path := findFirstPath(rootChild1, rootChild2)
	if path != nil {
		t.Fatalf("Wrong path found: %#+v", path)
	}

	// Find first path
	path = findFirstPath(root, rootChild2Child1Child1)
	if len(path) != 4 || path[0].ID != root.ID || path[1].ID != rootChild2.ID || path[2].ID != rootChild2Child1.ID || path[3].ID != rootChild2Child1Child1.ID {
		t.Fatalf("Wrong path found: %#+v", path)
	}

	// Get leaf node
	leaf := testGraph.getNextLeaf(root)
	if leaf.ID != rootChild1.ID {
		t.Fatalf("GetLeaf1: Got id %s, expected %s", leaf.ID, rootChild1.ID)
	}

	err = testGraph.addEdge("NotThere", leaf.ID)
	if err == nil {
		t.Fatal("No error when adding an edge from a non-existing node")
	}

	err = testGraph.addEdge(leaf.ID, "NotThere")
	if err == nil {
		t.Fatal("No error when adding an edge to a non-existing node")
	}

	// Remove node
	err = testGraph.removeNode(leaf.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Get leaf node
	leaf = testGraph.getNextLeaf(root)
	if leaf.ID != rootChild2Child1Child1.ID {
		t.Fatalf("GetLeaf2: Got id %s, expected %s", leaf.ID, rootChild2Child1Child1.ID)
	}

	// Remove node
	err = testGraph.removeNode(root.ID)
	if err == nil {
		t.Fatal("Expected error")
	}
}
