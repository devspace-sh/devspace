package util

import (
	"errors"
	"testing"
	jujuerr "github.com/juju/errors"
)

func TestJuju(t *testing.T) {
	err := funcThatCallsFailingFunc()

	if err == nil {
		t.Fail()
	} else {
		t.Log("Error: " + err.Error())
		t.Log("Cause: " + jujuerr.Cause(err).Error())
		t.Log("Details: " + jujuerr.Details(err))
		t.Log("StackTrace: " + jujuerr.ErrorStack(err))
	}
}

func funcThatCallsFailingFunc() error{
	err := failingFunc()
	return jujuerr.Trace(err)
}

func failingFunc() error {
	return errors.New("Some error")
}
