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

	//The same with ints
	set1[1] = 1
	set2[2] = 2

	set1[3] = 1
	set2[3] = 2

	set1[4] = 3
	set2[4] = 3

	//Now we use pointers as keys
	set1Exclusive := "set1Exclusive"
	set2Exclusive := "set2Exclusive"
	sameKeyDifferentVal := "sameKeyDifferentVal"
	sameKeySameVal := "sameKeySameVal"
	equalString1 := "equalString"
	equalString2 := "equalString"

	set1[&set1Exclusive] = "set1Exclusive"
	set2[&set2Exclusive] = "set2Exclusive"

	set1[&sameKeyDifferentVal] = "set1Val"
	set2[&sameKeyDifferentVal] = "set2Val"

	set1[&sameKeySameVal] = "commonVal"
	set2[&sameKeySameVal] = "commonVal"

	set1[&equalString1] = "set1Val"
	set2[&equalString2] = "set2Val"

	set1.MergeInto(set2)

	assert.Equal(t, set1["set1Exclusive"], "set1Exclusive")
	assert.Equal(t, set1["set2Exclusive"], "set2Exclusive")
	assert.Equal(t, set1["sameKeyDifferentVal"], "set2Val")
	assert.Equal(t, set1["sameKeySameVal"], "commonVal")

	assert.Equal(t, set1[1], 1)
	assert.Equal(t, set1[2], 2)
	assert.Equal(t, set1[3], 2)
	assert.Equal(t, set1[4], 3)

	assert.Equal(t, set1[&set1Exclusive], "set1Exclusive")
	assert.Equal(t, set1[&set2Exclusive], "set2Exclusive")
	assert.Equal(t, set1[&sameKeyDifferentVal], "set2Val")
	assert.Equal(t, set1[&sameKeySameVal], "commonVal")
	assert.Equal(t, set1[&equalString1], "set1Val")
	assert.Equal(t, set1[&equalString2], "set2Val")
}
