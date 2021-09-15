package ptr

import (
	"testing"

	"gotest.tools/assert"
)

func TestString(t *testing.T) {
	strPointer := String("hello")
	assert.Equal(t, *strPointer, "hello", "Returned string pointer points wrong")
}

func TestReverseString(t *testing.T) {
	str := "hello"
	strPointer := &str
	assert.Equal(t, str, ReverseString(strPointer), "Returned string is wrong")

	str = "bye"
	assert.Equal(t, str, ReverseString(strPointer), "Returned string is wrong")

	assert.Equal(t, "", ReverseString(nil), "Returned string is wrong")
}

func TestInt(t *testing.T) {
	intPointer := Int(1)
	assert.Equal(t, *intPointer, 1, "Returned int pointer points wrong")
}

func TestInt32(t *testing.T) {
	intPointer := Int32(int32(1))
	assert.Equal(t, *intPointer, int32(1), "Returned int32 pointer points wrong")
}

func TestInt64(t *testing.T) {
	intPointer := Int64(int64(1))
	assert.Equal(t, *intPointer, int64(1), "Returned int64 pointer points wrong")
}

func TestBool(t *testing.T) {
	boolPointer := Bool(true)
	assert.Equal(t, *boolPointer, true, "Returned bool pointer points wrong")

	boolPointer = Bool(false)
	assert.Equal(t, *boolPointer, false, "Returned bool pointer points wrong")
}

func TestReverseBool(t *testing.T) {
	someBool := true
	boolPointer := &someBool
	assert.Equal(t, ReverseBool(boolPointer), true, "Returned bool is wrong")

	someBool = false
	assert.Equal(t, ReverseBool(boolPointer), false, "Returned bool is wrong")

	assert.Equal(t, ReverseBool(nil), false, "Returned bool of nil is wrong")
}
