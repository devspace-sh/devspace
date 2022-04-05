package merge

import (
	"testing"

	"gotest.tools/assert"
)

func TestMerge(t *testing.T) {

	set1 := make(Values)
	set2 := make(Values)

	//Test data with strings
	set1["set1Exclusive"] = "set1Exclusive"
	set2["set2Exclusive"] = "set2Exclusive"

	set1["sameKeyDifferentVal"] = "set1Val"
	set2["sameKeyDifferentVal"] = "set2Val"

	set1["sameKeySameVal"] = "commonVal"
	set2["sameKeySameVal"] = "commonVal"

	set1.MergeInto(set2)

	assert.Equal(t, set1["set1Exclusive"], "set1Exclusive")
	assert.Equal(t, set1["set2Exclusive"], "set2Exclusive")
	assert.Equal(t, set1["sameKeyDifferentVal"], "set2Val")
	assert.Equal(t, set1["sameKeySameVal"], "commonVal")
}
