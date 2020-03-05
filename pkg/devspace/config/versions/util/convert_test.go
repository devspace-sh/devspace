package util

import (
	"testing"

	"gotest.tools/assert"
)

type struct1 struct {
	Hello            string
	Struct1Exclusive string
}

type struct2 struct {
	Hallo            string `yaml:"hello,omitempty"`
	Struct2Exclusive string
}

func TestConvert(t *testing.T) {
	o1 := &struct1{
		Hello:            "world",
		Struct1Exclusive: "TryToConvertThis",
	}
	o2 := &struct2{}
	err := Convert(o1, o2)
	assert.NilError(t, err, "Error converting valid values: %v")
	assert.Equal(t, o1.Hello, "world", "Conversion source altered")
	assert.Equal(t, o2.Hallo, "world", "No conversion done")
	assert.Equal(t, o2.Struct2Exclusive, "", "Conversion altered target exclusive field")
}
